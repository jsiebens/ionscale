package handlers

import (
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/labstack/echo/v4"
	"net"
	"net/http"
)

func HttpRedirectHandler(tls config.Tls) echo.HandlerFunc {
	if tls.Disable {
		return IndexHandler(http.StatusNotFound)
	}

	return func(c echo.Context) error {
		r := c.Request()
		toURL := "https://"
		requestHost := hostOnly(r.Host)
		toURL += requestHost
		toURL += r.URL.RequestURI()
		return c.Redirect(http.StatusMovedPermanently, toURL)
	}
}

func hostOnly(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}
