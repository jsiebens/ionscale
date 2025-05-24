package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/hashicorp/go-plugin"
	"github.com/jsiebens/ionscale/internal/auth"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/core"
	"github.com/jsiebens/ionscale/internal/database"
	"github.com/jsiebens/ionscale/internal/derp"
	"github.com/jsiebens/ionscale/internal/dns"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/handlers"
	"github.com/jsiebens/ionscale/internal/service"
	"github.com/jsiebens/ionscale/internal/stunserver"
	"github.com/jsiebens/ionscale/internal/templates"
	"github.com/jsiebens/ionscale/internal/util"
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
	"os"
	"os/signal"
	"syscall"
	"tailscale.com/types/key"
	"time"
)

func Start(ctx context.Context, c *config.Config) error {
	ctx = contextWithSigterm(ctx)

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

	util.EnsureIDProvider()

	derpMap, err := derp.LoadDERPSources(c)
	if err != nil {
		logger.Warn("not all derp sources are read successfully", zap.Error(err))
	}

	domain.SetDefaultDERPMap(derpMap)

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

	// prepare CertMagic
	if c.Tls.AcmeEnabled {
		storage, err := certmagicsql.NewStorage(ctx, db, certmagicsql.Options{})
		if err != nil {
			return err
		}

		certmagicLogger := logger.Named("certmagic")
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = c.Tls.AcmeEmail
		certmagic.DefaultACME.CA = c.Tls.AcmeCA
		certmagic.DefaultACME.Logger = certmagicLogger
		certmagic.Default.Logger = certmagicLogger
		certmagic.Default.Storage = storage

		cfg := certmagic.NewDefault()
		if err := cfg.ManageAsync(ctx, []string{c.PublicUrl.Hostname()}); err != nil {
			return logError(err)
		}
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

	createPeerHandler := func(machinePublicKey key.MachinePublic) http.Handler {
		registrationHandlers := handlers.NewRegistrationHandlers(machinePublicKey, c, sessionManager, repository)
		pollNetMapHandler := handlers.NewPollNetMapHandler(machinePublicKey, sessionManager, repository)
		dnsHandlers := handlers.NewDNSHandlers(machinePublicKey, dnsProvider)
		idTokenHandlers := handlers.NewIDTokenHandlers(machinePublicKey, c, repository)
		sshActionHandlers := handlers.NewSSHActionHandlers(machinePublicKey, c, repository)
		queryFeatureHandlers := handlers.NewQueryFeatureHandlers(machinePublicKey, dnsProvider, repository)
		updateHealthHandlers := handlers.NewUpdateHealthHandlers(machinePublicKey, repository)

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
		e.POST("/machine/update-health", updateHealthHandlers.UpdateHealth)

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

	metricsMux := echo.New()
	metricsMux.GET("/metrics", echoprometheus.NewHandler())
	pprof.Register(metricsMux)

	webMux := echo.New()
	webMux.Renderer = &templates.Renderer{}
	webMux.Pre(handlers.HttpsRedirect(c.Tls))
	webMux.Use(promMiddleware, EchoLogger(httpLogger), EchoErrorHandler(), EchoRecover())

	webMux.Any("/*", handlers.IndexHandler(http.StatusNotFound))
	webMux.Any("/", handlers.IndexHandler(http.StatusOK))
	webMux.POST(rpcPath+"*", echo.WrapHandler(rpcHandler))
	webMux.GET("/version", handlers.Version)
	webMux.GET("/key", handlers.KeyHandler(serverKey))
	webMux.POST("/ts2021", noiseHandlers.Upgrade)
	webMux.GET("/.well-known/jwks", oidcConfigHandlers.Jwks)
	webMux.GET("/.well-known/openid-configuration", oidcConfigHandlers.OpenIDConfig)

	csrf := middleware.CSRFWithConfig(middleware.CSRFConfig{TokenLookup: "form:_csrf"})
	webMux.GET("/a/:flow/:key", authenticationHandlers.StartAuth, csrf)
	webMux.POST("/a/:flow/:key", authenticationHandlers.ProcessAuth, csrf)
	webMux.GET("/a/callback", authenticationHandlers.Callback, csrf)
	webMux.POST("/a/callback", authenticationHandlers.EndAuth, csrf)
	webMux.GET("/a/success", authenticationHandlers.Success, csrf)
	webMux.GET("/a/error", authenticationHandlers.Error, csrf)

	if !c.DERP.Server.Disabled {
		derpHandlers := handlers.NewDERPHandler()

		metricsMux.GET("/debug/derp/traffic", derpHandlers.DebugTraffic)
		metricsMux.GET("/debug/derp/check", derpHandlers.DebugCheck)

		webMux.GET("/derp", derpHandlers.Handler)
		webMux.GET("/derp/latency-check", derpHandlers.LatencyCheck)
	}

	webL, err := webListener(c)
	if err != nil {
		return logError(err)
	}

	metricsL, err := metricsListener(c)
	if err != nil {
		return logError(err)
	}

	stunL, err := stunListener(c)
	if err != nil {
		return logError(err)
	}

	errorLog, err := zap.NewStdLogAt(logger, zap.DebugLevel)
	if err != nil {
		return logError(err)
	}

	webServer := &http.Server{ErrorLog: errorLog, Handler: h2c.NewHandler(webMux, &http2.Server{})}
	metricsServer := &http.Server{ErrorLog: errorLog, Handler: metricsMux}
	stunServer := stunserver.New(stunL)

	g, gCtx := errgroup.WithContext(ctx)

	go func() {
		<-gCtx.Done()
		logger.Sugar().Infow("Shutting down ionscale server")
		plugin.CleanupClients()
		shutdownHttpServer(metricsServer)
		shutdownHttpServer(webServer)
		_ = stunServer.Shutdown()
	}()

	g.Go(func() error { return serveHttp(webServer, webL) })
	g.Go(func() error { return serveHttp(metricsServer, metricsL) })
	g.Go(func() error { return stunServer.Serve() })

	fields := []zap.Field{
		zap.String("url", c.PublicUrl.String()),
		zap.String("addr", c.ListenAddr),
		zap.String("metrics_addr", c.MetricsListenAddr),
	}

	if !c.DERP.Server.Disabled {
		fields = append(fields, zap.String("stun_addr", c.StunListenAddr))
	} else {
		logger.Warn("Embedded DERP is disabled")
	}

	if c.Tls.AcmeEnabled {
		logger.Info("TLS is enabled with ACME", zap.String("domain", c.PublicUrl.Hostname()))
		logger.Info("Server is running", fields...)
	} else if !c.Tls.Disable {
		logger.Info("TLS is enabled", zap.String("cert", c.Tls.CertFile))
		logger.Info("Server is running", fields...)
	} else {
		logger.Warn("TLS is disabled")
		logger.Info("Server is running", fields...)
	}

	return g.Wait()
}

func serveHttp(s *http.Server, l net.Listener) error {
	if l == nil || s == nil {
		return nil
	}
	if err := s.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func shutdownHttpServer(s *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Shutdown(ctx)
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

func webListener(config *config.Config) (net.Listener, error) {
	if config.Tls.Disable {
		return net.Listen("tcp", config.ListenAddr)
	}

	if config.Tls.AcmeEnabled {
		cfg := certmagic.NewDefault()
		tlsConfig := cfg.TLSConfig()
		tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)
		return tls.Listen("tcp", config.ListenAddr, tlsConfig)
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

	return tls.Listen("tcp", config.ListenAddr, tlsConfig)
}

func metricsListener(config *config.Config) (net.Listener, error) {
	return net.Listen("tcp", config.MetricsListenAddr)
}

func stunListener(config *config.Config) (*net.UDPConn, error) {
	if config.DERP.Server.Disabled {
		return nil, nil
	}

	addr, err := net.ResolveUDPAddr("udp", config.StunListenAddr)
	if err != nil {
		return nil, err
	}

	return net.ListenUDP("udp", addr)
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

func contextWithSigterm(ctx context.Context) context.Context {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-signalCh:
		case <-ctx.Done():
		}
	}()

	return ctxWithCancel
}
