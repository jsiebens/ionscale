package handlers

import (
	"github.com/caddyserver/certmagic"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net"
	"net/http"
)

func httpsRedirectSkipper(c config.Tls) func(ctx echo.Context) bool {
	return func(ctx echo.Context) bool {
		if ctx.Request().Method == "POST" && ctx.Request().RequestURI == "/ts2021" {
			return true
		}
		return !c.ForceHttps
	}
}

func HttpsRedirect(c config.Tls) echo.MiddlewareFunc {
	return middleware.HTTPSRedirectWithConfig(middleware.RedirectConfig{
		Skipper: httpsRedirectSkipper(c),
	})
}

func HttpRedirectHandler(tls config.Tls) echo.HandlerFunc {
	if tls.Disable {
		return IndexHandler(http.StatusNotFound)
	}

	if tls.AcmeEnabled {
		cfg := certmagic.NewDefault()
		if len(cfg.Issuers) > 0 {
			if am, ok := cfg.Issuers[0].(*certmagic.ACMEIssuer); ok {
				handler := am.HTTPChallengeHandler(http.HandlerFunc(httpRedirectHandler))
				return echo.WrapHandler(handler)
			}
		}
	}

	return echo.WrapHandler(http.HandlerFunc(httpRedirectHandler))
}

func httpRedirectHandler(w http.ResponseWriter, r *http.Request) {
	toURL := "https://"
	requestHost := hostOnly(r.Host)
	toURL += requestHost
	toURL += r.URL.RequestURI()
	w.Header().Set("Connection", "close")
	http.Redirect(w, r, toURL, http.StatusMovedPermanently)
}

func hostOnly(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}
