package server

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/jsiebens/ionscale/internal/errors"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
	"time"
)

func EchoErrorHandler(logger hclog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()

			if err := next(c); err != nil {
				switch t := err.(type) {
				case *echo.HTTPError:
					return err
				case *errors.Error:
					logger.Error("error processing request",
						"err", t.Cause,
						"location", t.Location,
						"http.method", request.Method,
						"http.uri", request.RequestURI,
					)
				default:
					logger.Error("error processing request",
						"err", err,
						"http.method", request.Method,
						"http.uri", request.RequestURI,
					)
				}

				if strings.HasPrefix(request.RequestURI, "/a/") {
					return c.Render(http.StatusInternalServerError, "error.html", nil)
				}
			}

			return nil
		}
	}
}

func EchoLogger(logger hclog.Logger) echo.MiddlewareFunc {
	httpLogger := logger.Named("http")
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if !httpLogger.IsTrace() {
				return next(c)
			}

			request := c.Request()
			response := c.Response()
			start := time.Now()
			if err = next(c); err != nil {
				c.Error(err)
			}

			httpLogger.Trace("finished server http call",
				"http.code", response.Status,
				"http.method", request.Method,
				"http.uri", request.RequestURI,
				"http.start_time", start.Format(time.RFC3339),
				"http.duration", time.Since(start))

			return
		}
	}
}

func EchoRecover(logger hclog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apply := func() (topErr error) {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						topErr = err
					}
				}()
				return next(c)
			}
			return apply()
		}
	}
}

func ErrorRedirect() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("redirect_on_error", true)
			return next(c)
		}
	}
}

func EchoMetrics(p *prometheus.Prometheus) echo.MiddlewareFunc {
	return p.HandlerFunc
}
