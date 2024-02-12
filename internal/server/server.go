package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/jsiebens/ionscale/internal/auth"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/core"
	"github.com/jsiebens/ionscale/internal/database"
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/handlers"
	"github.com/jsiebens/ionscale/internal/service"
	"github.com/jsiebens/ionscale/internal/templates"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	certmagicsql "github.com/travisjeffery/certmagic-sqlstorage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
	"net/url"
	"os"
	"tailscale.com/types/key"
)

func Start(ctx context.Context, c *config.Config) error {
	logger, err := setupLogging(c.Logging)
	if err != nil {
		return err
	}

	logger.Info("Starting ionscale server")

	logError := func(err error) error {
		if err != nil {
			zap.L().WithOptions(zap.AddCallerSkip(1)).Error("Unable to start server", zap.Error(err))
		}
		return err
	}

	httpLogger := logger.Named("http")
	dbLogger := logger.Named("db")

	db, repository, err := database.OpenDB(&c.Database, dbLogger)
	if err != nil {
		return logError(err)
	}

	sessionManager := core.NewPollMapSessionManager()

	defaultControlKeys, err := repository.GetControlKeys(ctx)
	if err != nil {
		return logError(err)
	}

	serverKey, err := c.ReadServerKeys(defaultControlKeys)
	if err != nil {
		return logError(err)
	}

	core.StartWorker(repository, sessionManager)

	serverUrl, err := url.Parse(c.ServerUrl)
	if err != nil {
		return logError(err)
	}

	// prepare CertMagic
	if c.Tls.AcmeEnabled {
		storage, err := certmagicsql.NewStorage(ctx, db, certmagicsql.Options{})
		if err != nil {
			return err
		}

		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = c.Tls.AcmeEmail
		certmagic.DefaultACME.CA = c.Tls.AcmeCA
		certmagic.Default.Logger = logger.Named("certmagic")
		certmagic.Default.Storage = storage

		cfg := certmagic.NewDefault()
		if err := cfg.ManageAsync(ctx, []string{serverUrl.Host}); err != nil {
			return logError(err)
		}

		c.HttpListenAddr = fmt.Sprintf(":%d", certmagic.HTTPPort)
		c.HttpsListenAddr = fmt.Sprintf(":%d", certmagic.HTTPSPort)
	}

	authProvider, systemIAMPolicy, err := setupAuthProvider(c.Auth)
	if err != nil {
		return logError(fmt.Errorf("error configuring OIDC provider: %v", err))
	}

	dnsProvider, err := dns.NewProvider(c.DNS)
	if err != nil {
		return logError(err)
	}

	promMiddleware := echoprometheus.NewMiddleware("http")

	metricsHandler := echo.New()
	metricsHandler.GET("/metrics", echoprometheus.NewHandler())
	pprof.Register(metricsHandler)

	createPeerHandler := func(machinePublicKey key.MachinePublic) http.Handler {
		registrationHandlers := handlers.NewRegistrationHandlers(machinePublicKey, c, sessionManager, repository)
		pollNetMapHandler := handlers.NewPollNetMapHandler(machinePublicKey, sessionManager, repository)
		dnsHandlers := handlers.NewDNSHandlers(machinePublicKey, dnsProvider)
		idTokenHandlers := handlers.NewIDTokenHandlers(machinePublicKey, c, repository)
		sshActionHandlers := handlers.NewSSHActionHandlers(machinePublicKey, c, repository)
		queryFeatureHandlers := handlers.NewQueryFeatureHandlers(machinePublicKey, dnsProvider, repository)

		e := echo.New()
		e.Binder = handlers.JsonBinder{}
		e.Use(promMiddleware, EchoLogger(httpLogger), EchoErrorHandler(), EchoRecover())
		e.POST("/machine/register", registrationHandlers.Register)
		e.POST("/machine/map", pollNetMapHandler.PollNetMap)
		e.POST("/machine/set-dns", dnsHandlers.SetDNS)
		e.POST("/machine/id-token", idTokenHandlers.FetchToken)
		e.GET("/machine/ssh/action/:src_machine_id/to/:dst_machine_id", sshActionHandlers.StartAuth)
		e.GET("/machine/ssh/action/:src_machine_id/to/:dst_machine_id/:check_period", sshActionHandlers.StartAuth)
		e.GET("/machine/ssh/action/check/:key", sshActionHandlers.CheckAuth)
		e.POST("/machine/feature/query", queryFeatureHandlers.QueryFeature)

		return e
	}

	noiseHandlers := handlers.NewNoiseHandlers(serverKey.ControlKey, createPeerHandler)
	oidcConfigHandlers := handlers.NewOIDCConfigHandlers(c, repository)

	authenticationHandlers := handlers.NewAuthenticationHandlers(
		c,
		authProvider,
		systemIAMPolicy,
		repository,
	)

	rpcService := service.NewService(c, authProvider, dnsProvider, repository, sessionManager)
	rpcPath, rpcHandler := NewRpcHandler(serverKey.SystemAdminKey, repository, rpcService)

	nonTlsAppHandler := echo.New()
	nonTlsAppHandler.Use(promMiddleware, EchoLogger(httpLogger), EchoErrorHandler(), EchoRecover())
	nonTlsAppHandler.POST("/ts2021", noiseHandlers.Upgrade)
	nonTlsAppHandler.Any("/*", handlers.HttpRedirectHandler(c.Tls))

	tlsAppHandler := echo.New()
	tlsAppHandler.Renderer = &templates.Renderer{}
	tlsAppHandler.Pre(handlers.HttpsRedirect(c.Tls))
	tlsAppHandler.Use(promMiddleware, EchoLogger(httpLogger), EchoErrorHandler(), EchoRecover())

	tlsAppHandler.Any("/*", handlers.IndexHandler(http.StatusNotFound))
	tlsAppHandler.Any("/", handlers.IndexHandler(http.StatusOK))
	tlsAppHandler.POST(rpcPath+"*", echo.WrapHandler(rpcHandler))
	tlsAppHandler.GET("/version", handlers.Version)
	tlsAppHandler.GET("/key", handlers.KeyHandler(serverKey))
	tlsAppHandler.POST("/ts2021", noiseHandlers.Upgrade)
	tlsAppHandler.GET("/.well-known/jwks", oidcConfigHandlers.Jwks)
	tlsAppHandler.GET("/.well-known/openid-configuration", oidcConfigHandlers.OpenIDConfig)

	csrf := middleware.CSRFWithConfig(middleware.CSRFConfig{TokenLookup: "form:_csrf"})
	tlsAppHandler.GET("/a/:flow/:key", authenticationHandlers.StartAuth, csrf)
	tlsAppHandler.POST("/a/:flow/:key", authenticationHandlers.ProcessAuth, csrf)
	tlsAppHandler.GET("/a/callback", authenticationHandlers.Callback, csrf)
	tlsAppHandler.POST("/a/callback", authenticationHandlers.EndAuth, csrf)
	tlsAppHandler.GET("/a/success", authenticationHandlers.Success, csrf)
	tlsAppHandler.GET("/a/error", authenticationHandlers.Error, csrf)

	tlsL, err := tlsListener(c)
	if err != nil {
		return logError(err)
	}

	nonTlsL, err := nonTlsListener(c)
	if err != nil {
		return logError(err)
	}

	metricsL, err := metricsListener(c)
	if err != nil {
		return logError(err)
	}

	httpL := selectListener(tlsL, nonTlsL)
	http2Server := &http2.Server{}
	g := new(errgroup.Group)

	g.Go(func() error { return http.Serve(httpL, h2c.NewHandler(tlsAppHandler, http2Server)) })
	g.Go(func() error { return http.Serve(metricsL, metricsHandler) })

	if tlsL != nil {
		g.Go(func() error { return http.Serve(nonTlsL, nonTlsAppHandler) })
	}

	if c.Tls.AcmeEnabled {
		logger.Sugar().Infow("TLS is enabled with ACME", "domain", serverUrl.Host)
		logger.Sugar().Infow("Server is running", "http_addr", c.HttpListenAddr, "https_addr", c.HttpsListenAddr, "metrics_addr", c.MetricsListenAddr)
	} else if !c.Tls.Disable {
		logger.Sugar().Infow("TLS is enabled", "cert", c.Tls.CertFile)
		logger.Sugar().Infow("Server is running", "http_addr", c.HttpListenAddr, "https_addr", c.HttpsListenAddr, "metrics_addr", c.MetricsListenAddr)
	} else {
		logger.Sugar().Warnw("TLS is disabled")
		logger.Sugar().Infow("Server is running", "http_addr", c.HttpListenAddr, "metrics_addr", c.MetricsListenAddr)
	}

	return g.Wait()
}

func setupAuthProvider(config config.Auth) (auth.Provider, *domain.IAMPolicy, error) {
	if len(config.Provider.Issuer) == 0 {
		return nil, &domain.IAMPolicy{}, nil
	}

	authProvider, err := auth.NewOIDCProvider(&config.Provider)
	if err != nil {
		return nil, nil, err
	}

	return authProvider, &domain.IAMPolicy{
		Subs:    config.SystemAdminPolicy.Subs,
		Emails:  config.SystemAdminPolicy.Emails,
		Filters: config.SystemAdminPolicy.Filters,
	}, nil
}

func metricsListener(config *config.Config) (net.Listener, error) {
	return net.Listen("tcp", config.MetricsListenAddr)
}

func tlsListener(config *config.Config) (net.Listener, error) {
	if config.Tls.Disable {
		return nil, nil
	}

	if config.Tls.AcmeEnabled {
		cfg := certmagic.NewDefault()
		tlsConfig := cfg.TLSConfig()
		tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)
		return tls.Listen("tcp", config.HttpsListenAddr, tlsConfig)
	}

	certPEMBlock, err := os.ReadFile(config.Tls.CertFile)
	if err != nil {
		return nil, fmt.Errorf("error reading cert file: %v", err)
	}
	keyPEMBlock, err := os.ReadFile(config.Tls.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("error reading key file: %v", err)
	}

	cer, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, fmt.Errorf("error reading cert and key file: %v", err)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}}

	return tls.Listen("tcp", config.HttpsListenAddr, tlsConfig)
}

func nonTlsListener(config *config.Config) (net.Listener, error) {
	return net.Listen("tcp", config.HttpListenAddr)
}

func selectListener(a net.Listener, b net.Listener) net.Listener {
	if a != nil {
		return a
	}
	return b
}

func setupLogging(config config.Logging) (*zap.Logger, error) {
	level, err := zap.ParseAtomicLevel(config.Level)
	if err != nil {
		return nil, err
	}

	pc := zap.NewProductionConfig()
	pc.Level = level
	pc.DisableStacktrace = true
	pc.OutputPaths = []string{"stdout"}
	pc.Encoding = "console"
	pc.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	pc.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if config.File != "" {
		pc.OutputPaths = []string{config.File}
	}

	if config.Format == "json" {
		pc.Encoding = "json"
	}

	logger, err := pc.Build()
	if err != nil {
		return nil, err
	}

	globalLogger := logger.Named("ionscale")
	zap.ReplaceGlobals(globalLogger)

	return globalLogger, nil
}
