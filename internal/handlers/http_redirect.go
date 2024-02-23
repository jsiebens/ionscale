package handlers

import (
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
