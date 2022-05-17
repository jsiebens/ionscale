package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/caddyserver/certmagic"
	"github.com/hashicorp/go-hclog"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/database"
	"github.com/jsiebens/ionscale/internal/handlers"
	"github.com/jsiebens/ionscale/internal/service"
	"github.com/jsiebens/ionscale/internal/templates"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	echo_prometheus "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"tailscale.com/types/key"
	"time"
)

func Start(config *config.Config) error {
	logger, err := setupLogging(config.Logging)
	if err != nil {
		return err
	}

	logger.Info("Starting ionscale server")

	serverKey, err := config.ReadServerKeys()
	if err != nil {
		return err
	}

	_, repository, err := database.OpenDB(&config.Database, logger)
	if err != nil {
		return err
	}

	controlKeys, err := repository.GetControlKeys(context.Background())
	if err != nil {
		return err
	}

	pendingMachineRegistrationRequests := cache.New(5*time.Minute, 10*time.Minute)
	brokers := broker.NewBrokerPool()
	offlineTimers := handlers.NewOfflineTimers(repository, brokers)
	reaper := handlers.NewReaper(brokers, repository)

	go offlineTimers.Start()
	go reaper.Start()

	// prepare CertMagic
	if config.Tls.CertMagicDomain != "" {
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = config.Tls.CertMagicEmail
		certmagic.DefaultACME.CA = config.Tls.CertMagicCA
		if config.Tls.CertMagicStoragePath != "" {
			certmagic.Default.Storage = &certmagic.FileStorage{Path: config.Tls.CertMagicStoragePath}
		}

		cfg := certmagic.NewDefault()
		if err := cfg.ManageSync(context.Background(), []string{config.Tls.CertMagicDomain}); err != nil {
			return err
		}

		config.HttpListenAddr = fmt.Sprintf(":%d", certmagic.HTTPPort)
		config.HttpsListenAddr = fmt.Sprintf(":%d", certmagic.HTTPSPort)
	}

	createPeerHandler := func(p key.MachinePublic) http.Handler {
		registrationHandlers := handlers.NewRegistrationHandlers(bind.DefaultBinder(p), config, repository, pendingMachineRegistrationRequests)
		pollNetMapHandler := handlers.NewPollNetMapHandler(bind.DefaultBinder(p), brokers, repository, offlineTimers)

		e := echo.New()
		e.Use(EchoLogger(logger))
		e.Use(EchoRecover(logger))
		e.POST("/machine/register", registrationHandlers.Register)
		e.POST("/machine/map", pollNetMapHandler.PollNetMap)

		return e
	}

	noiseHandlers := handlers.NewNoiseHandlers(controlKeys.ControlKey, createPeerHandler)
	registrationHandlers := handlers.NewRegistrationHandlers(bind.BoxBinder(controlKeys.LegacyControlKey), config, repository, pendingMachineRegistrationRequests)
	pollNetMapHandler := handlers.NewPollNetMapHandler(bind.BoxBinder(controlKeys.LegacyControlKey), brokers, repository, offlineTimers)
	authenticationHandlers := handlers.NewAuthenticationHandlers(
		config,
		repository,
		pendingMachineRegistrationRequests,
	)

	p := echo_prometheus.NewPrometheus("http", nil)

	metricsHandler := echo.New()
	p.SetMetricsPath(metricsHandler)

	nonTlsAppHandler := echo.New()
	nonTlsAppHandler.Use(EchoRecover(logger))
	nonTlsAppHandler.Use(EchoLogger(logger))
	nonTlsAppHandler.Use(p.HandlerFunc)
	nonTlsAppHandler.POST("/ts2021", noiseHandlers.Upgrade)
	nonTlsAppHandler.Any("/*", handlers.HttpRedirectHandler(config.Tls))

	tlsAppHandler := echo.New()
	tlsAppHandler.Renderer = templates.NewTemplates()
	tlsAppHandler.Use(EchoRecover(logger))
	tlsAppHandler.Use(EchoLogger(logger))
	tlsAppHandler.Use(p.HandlerFunc)

	tlsAppHandler.Any("/*", handlers.IndexHandler(http.StatusNotFound))
	tlsAppHandler.Any("/", handlers.IndexHandler(http.StatusOK))
	tlsAppHandler.GET("/version", handlers.Version)
	tlsAppHandler.GET("/key", handlers.KeyHandler(controlKeys))
	tlsAppHandler.POST("/ts2021", noiseHandlers.Upgrade)
	tlsAppHandler.POST("/machine/:id", registrationHandlers.Register)
	tlsAppHandler.POST("/machine/:id/map", pollNetMapHandler.PollNetMap)

	auth := tlsAppHandler.Group("/a")
	auth.GET("/:key", authenticationHandlers.StartAuth)
	auth.POST("/:key", authenticationHandlers.StartAuth)
	auth.GET("/success", authenticationHandlers.Success)
	auth.GET("/error", authenticationHandlers.Error)

	grpcService := service.NewService(repository, brokers)
	grpcServer := NewGrpcServer(logger, serverKey.SystemAdminKey)
	api.RegisterIonscaleServer(grpcServer, grpcService)

	tlsL, err := tlsListener(config)
	if err != nil {
		return err
	}

	nonTlsL, err := nonTlsListener(config)
	if err != nil {
		return err
	}

	metricsL, err := metricsListener(config)
	if err != nil {
		return err
	}

	mux := cmux.New(selectListener(tlsL, nonTlsL))
	grpcL := mux.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"),
		cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc+proto"),
	)
	grpcWebL := mux.Match(cmux.HTTP1HeaderFieldPrefix("content-type", "application/grpc-web"))
	httpL := mux.Match(cmux.Any())

	grpcWebHandler := grpcweb.WrapServer(grpcServer)
	http2Server := &http2.Server{}
	g := new(errgroup.Group)

	g.Go(func() error { return grpcServer.Serve(grpcL) })
	g.Go(func() error { return http.Serve(grpcWebL, h2c.NewHandler(grpcWebHandler, http2Server)) })
	g.Go(func() error { return http.Serve(httpL, h2c.NewHandler(tlsAppHandler, http2Server)) })
	g.Go(func() error { return http.Serve(metricsL, metricsHandler) })
	g.Go(func() error { return mux.Serve() })

	if tlsL != nil {
		g.Go(func() error { return http.Serve(nonTlsL, nonTlsAppHandler) })
	}

	if config.Tls.CertMagicDomain != "" {
		logger.Info("TLS is enabled with CertMagic", "domain", config.Tls.CertMagicDomain)
		logger.Info("Server is running", "http_addr", config.HttpListenAddr, "https_addr", config.HttpsListenAddr, "metrics_addr", config.MetricsListenAddr)
	} else if !config.Tls.Disable {
		logger.Info("TLS is enabled", "cert", config.Tls.CertFile)
		logger.Info("Server is running", "http_addr", config.HttpListenAddr, "https_addr", config.HttpsListenAddr, "metrics_addr", config.MetricsListenAddr)
	} else {
		logger.Warn("TLS is disabled")
		logger.Info("Server is running", "http_addr", config.HttpListenAddr, "metrics_addr", config.MetricsListenAddr)
	}

	return g.Wait()
}

func metricsListener(config *config.Config) (net.Listener, error) {
	return net.Listen("tcp", config.MetricsListenAddr)
}

func tlsListener(config *config.Config) (net.Listener, error) {
	if config.Tls.CertMagicDomain != "" {
		cfg := certmagic.NewDefault()
		tlsConfig := cfg.TLSConfig()
		tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)
		return tls.Listen("tcp", config.HttpsListenAddr, tlsConfig)
	}

	if !config.Tls.Disable {
		cer, err := tls.LoadX509KeyPair(config.Tls.CertFile, config.Tls.KeyFile)
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cer}}

		return tls.Listen("tcp", config.HttpsListenAddr, tlsConfig)
	}

	return nil, nil
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
