package server

import (
	"fmt"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

func EchoErrorHandler() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()

			err := next(c)

			if err != nil && strings.HasPrefix(request.RequestURI, "/a/") {
				return c.Render(http.StatusInternalServerError, "error.html", nil)
			}

			return err
		}
	}
}

func EchoLogger(logger *zap.Logger) echo.MiddlewareFunc {
	httpLogger := logger.Sugar()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if !httpLogger.Level().Enabled(zap.DebugLevel) {
				return next(c)
			}

			request := c.Request()
			response := c.Response()
			start := time.Now()
			if err = next(c); err != nil {
				c.Error(err)
			}

			httpLogger.Debugw("finished server http call",
				"http.code", response.Status,
				"http.method", request.Method,
				"http.uri", request.RequestURI,
				"http.start_time", start.Format(time.RFC3339),
				"http.duration", time.Since(start))

			return
		}
	}
}

func EchoRecover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apply := func() (topErr error) {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						zap.L().Error("panic when processing request", zap.Error(err))
						topErr = err
					}
				}()
				return next(c)
			}
			return apply()
		}
	}
}

func EchoMetrics(p *prometheus.Prometheus) echo.MiddlewareFunc {
	return p.HandlerFunc
}
