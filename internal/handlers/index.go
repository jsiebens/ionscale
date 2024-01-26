package handlers

import (
	tpl "github.com/jsiebens/ionscale/internal/templates"
	"github.com/jsiebens/ionscale/internal/version"
	"github.com/labstack/echo/v4"
)

func IndexHandler(code int) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(code, "", tpl.Index(version.GetReleaseInfo()))
	}
}
