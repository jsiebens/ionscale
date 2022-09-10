package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/hashicorp/go-hclog"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/database"
	"github.com/jsiebens/ionscale/internal/domain"
	"github.com/jsiebens/ionscale/internal/handlers"
	"github.com/jsiebens/ionscale/internal/provider"
	"github.com/jsiebens/ionscale/internal/service"
	"github.com/jsiebens/ionscale/internal/templates"
	echo_prometheus "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"tailscale.com/types/key"
)

func Start(c *config.Config) error {
	logger, err := setupLogging(c.Logging)
	if err != nil {
		return err
	}

	logger.Info("Starting ionscale server")

	serverKey, err := c.ReadServerKeys()
	if err != nil {
		return err
	}

	repository, brokers, err := database.OpenDB(&c.Database, logger)
	if err != nil {
		return err
	}

	offlineTimers := handlers.NewOfflineTimers(repository, brokers)
	reaper := handlers.NewReaper(brokers, repository)

	go offlineTimers.Start()
	go reaper.Start()

	serverUrl, err := url.Parse(c.ServerUrl)
	if err != nil {
		return err
	}

	// prepare CertMagic
	if c.Tls.AcmeEnabled {
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = c.Tls.AcmeEmail
		certmagic.DefaultACME.CA = c.Tls.AcmeCA
		if c.Tls.AcmePath != "" {
			certmagic.Default.Storage = &certmagic.FileStorage{Path: c.Tls.AcmePath}
		}

		cfg := certmagic.NewDefault()
		if err := cfg.ManageAsync(context.Background(), []string{serverUrl.Host}); err != nil {
			return err
		}

		c.HttpListenAddr = fmt.Sprintf(":%d", certmagic.HTTPPort)
		c.HttpsListenAddr = fmt.Sprintf(":%d", certmagic.HTTPSPort)
	}

	createPeerHandler := func(p key.MachinePublic) http.Handler {
		registrationHandlers := handlers.NewRegistrationHandlers(bind.DefaultBinder(p), c, brokers, repository)
		pollNetMapHandler := handlers.NewPollNetMapHandler(bind.DefaultBinder(p), brokers, repository, offlineTimers)

		e := echo.New()
		e.Use(EchoLogger(logger))
		e.Use(EchoRecover(logger))
		e.POST("/machine/register", registrationHandlers.Register)
		e.POST("/machine/map", pollNetMapHandler.PollNetMap)

		return e
	}

	authProvider, systemIAMPolicy, err := setupAuthProvider(c.AuthProvider)
	if err != nil {
		return err
	}

	noiseHandlers := handlers.NewNoiseHandlers(serverKey.ControlKey, createPeerHandler)
	registrationHandlers := handlers.NewRegistrationHandlers(bind.BoxBinder(serverKey.LegacyControlKey), c, brokers, repository)
	pollNetMapHandler := handlers.NewPollNetMapHandler(bind.BoxBinder(serverKey.LegacyControlKey), brokers, repository, offlineTimers)
	authenticationHandlers := handlers.NewAuthenticationHandlers(
		c,
		authProvider,
		systemIAMPolicy,
		repository,
	)

	rpcService := service.NewService(c, authProvider, repository, brokers)
	rpcPath, rpcHandler := NewRpcHandler(serverKey.SystemAdminKey, repository, rpcService)

	p := echo_prometheus.NewPrometheus("http", nil)

	metricsHandler := echo.New()
	p.SetMetricsPath(metricsHandler)

	nonTlsAppHandler := echo.New()
	nonTlsAppHandler.Use(EchoRecover(logger))
	nonTlsAppHandler.Use(EchoLogger(logger))
	nonTlsAppHandler.Use(p.HandlerFunc)
	nonTlsAppHandler.POST("/ts2021", noiseHandlers.Upgrade)
	nonTlsAppHandler.Any("/*", handlers.HttpRedirectHandler(c.Tls))

	tlsAppHandler := echo.New()
	tlsAppHandler.Pre(handlers.HttpsRedirect(c.Tls))
	tlsAppHandler.Renderer = templates.NewTemplates()
	tlsAppHandler.Use(EchoRecover(logger))
	tlsAppHandler.Use(EchoLogger(logger))
	tlsAppHandler.Use(p.HandlerFunc)

	tlsAppHandler.Any("/*", handlers.IndexHandler(http.StatusNotFound))
	tlsAppHandler.Any("/", handlers.IndexHandler(http.StatusOK))
	tlsAppHandler.POST(rpcPath+"*", echo.WrapHandler(rpcHandler))
	tlsAppHandler.GET("/version", handlers.Version)
	tlsAppHandler.GET("/key", handlers.KeyHandler(serverKey))
	tlsAppHandler.POST("/ts2021", noiseHandlers.Upgrade)
	tlsAppHandler.POST("/machine/:id", registrationHandlers.Register)
	tlsAppHandler.POST("/machine/:id/map", pollNetMapHandler.PollNetMap)

	auth := tlsAppHandler.Group("/a")
	auth.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup: "form:_csrf",
	}))
	auth.GET("/:key", authenticationHandlers.StartAuth)
	auth.POST("/:key", authenticationHandlers.ProcessAuth)
	auth.GET("/c/:key", authenticationHandlers.StartCliAuth)
	auth.GET("/callback", authenticationHandlers.Callback)
	auth.POST("/callback", authenticationHandlers.EndOAuth)
	auth.GET("/success", authenticationHandlers.Success)
	auth.GET("/error", authenticationHandlers.Error)

	tlsL, err := tlsListener(c)
	if err != nil {
		return err
	}

	nonTlsL, err := nonTlsListener(c)
	if err != nil {
		return err
	}

	metricsL, err := metricsListener(c)
	if err != nil {
		return err
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
		logger.Info("TLS is enabled with ACME", "domain", serverUrl.Host)
		logger.Info("Server is running", "http_addr", c.HttpListenAddr, "https_addr", c.HttpsListenAddr, "metrics_addr", c.MetricsListenAddr)
	} else if !c.Tls.Disable {
		logger.Info("TLS is enabled", "cert", c.Tls.CertFile)
		logger.Info("Server is running", "http_addr", c.HttpListenAddr, "https_addr", c.HttpsListenAddr, "metrics_addr", c.MetricsListenAddr)
	} else {
		logger.Warn("TLS is disabled")
		logger.Info("Server is running", "http_addr", c.HttpListenAddr, "metrics_addr", c.MetricsListenAddr)
	}

	return g.Wait()
}

func setupAuthProvider(config config.AuthProvider) (provider.AuthProvider, *domain.IAMPolicy, error) {
	if len(config.Issuer) == 0 {
		return nil, &domain.IAMPolicy{}, nil
	}

	authProvider, err := provider.NewOIDCProvider(&config)
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

	cer, err := tls.LoadX509KeyPair(config.Tls.CertFile, config.Tls.KeyFile)
	if err != nil {
		return nil, err
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

func setupLogging(config config.Logging) (hclog.Logger, error) {
	file, err := createLogFile(config)
	if err != nil {
		return nil, err
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:       "ionscale",
		Level:      hclog.LevelFromString(config.Level),
		JSONFormat: strings.ToLower(config.Format) == "json",
		Output:     file,
	})

	log.SetOutput(appLogger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true}))
	log.SetPrefix("")
	log.SetFlags(0)

	return appLogger, nil
}

func createLogFile(config config.Logging) (*os.File, error) {
	if config.File != "" {
		f, err := os.OpenFile(config.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
	return os.Stdout, nil
}
