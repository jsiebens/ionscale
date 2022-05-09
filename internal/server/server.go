package server

import (
	"github.com/hashicorp/go-hclog"
	"github.com/jsiebens/ionscale/internal/bind"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/database"
	"github.com/jsiebens/ionscale/internal/handlers"
	"github.com/jsiebens/ionscale/internal/mux"
	"github.com/jsiebens/ionscale/internal/service"
	"github.com/jsiebens/ionscale/internal/templates"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	echo_prometheus "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"log"
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

	_, repository, err := database.OpenDB(&config.Database, logger)
	if err != nil {
		return err
	}

	serverKey, err := config.ReadServerKeys()
	if err != nil {
		return err
	}

	pendingMachineRegistrationRequests := cache.New(5*time.Minute, 10*time.Minute)
	brokers := broker.NewBrokerPool()
	offlineTimers := handlers.NewOfflineTimers(repository, brokers)
	reaper := handlers.NewReaper(brokers, repository)

	go offlineTimers.Start()
	go reaper.Start()

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

	noiseHandlers := handlers.NewNoiseHandlers(serverKey.ControlKey, createPeerHandler)
	registrationHandlers := handlers.NewRegistrationHandlers(bind.BoxBinder(serverKey.LegacyControlKey), config, repository, pendingMachineRegistrationRequests)
	pollNetMapHandler := handlers.NewPollNetMapHandler(bind.BoxBinder(serverKey.LegacyControlKey), brokers, repository, offlineTimers)
	authenticationHandlers := handlers.NewAuthenticationHandlers(
		config,
		repository,
		pendingMachineRegistrationRequests,
	)

	p := echo_prometheus.NewPrometheus("http", nil)

	e := echo.New()
	e.Renderer = templates.NewTemplates()
	e.Use(EchoRecover(logger))
	e.Use(EchoLogger(logger))
	e.Use(p.HandlerFunc)

	m := echo.New()
	p.SetMetricsPath(m)

	e.Any("/*", handlers.IndexHandler(http.StatusNotFound))
	e.Any("/", handlers.IndexHandler(http.StatusOK))
	e.GET("/version", handlers.Version)
	e.GET("/key", handlers.KeyHandler(serverKey))
	e.POST("/ts2021", noiseHandlers.Upgrade)
	e.POST("/machine/:id", registrationHandlers.Register)
	e.POST("/machine/:id/map", pollNetMapHandler.PollNetMap)

	auth := e.Group("/a")
	auth.GET("/:key", authenticationHandlers.StartAuth)
	auth.POST("/:key", authenticationHandlers.StartAuth)
	auth.GET("/success", authenticationHandlers.Success)
	auth.GET("/error", authenticationHandlers.Error)

	grpcService := service.NewService(repository, brokers)
	grpcServer := NewGrpcServer(logger, serverKey.SystemAdminKey)
	api.RegisterIonscaleServer(grpcServer, grpcService)

	if config.Tls.Disable {
		logger.Warn("TLS is disabled")
	} else {
		logger.Info("TLS is enabled", "cert", config.Tls.CertFile)
	}

	logger.Info("Server is running", "addr", config.ListenAddr, "metrics", config.Metrics.ListenAddr)

	return mux.Serve(grpcServer, e, m, config)
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
