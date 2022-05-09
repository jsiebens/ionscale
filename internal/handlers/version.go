package handlers

import (
	"github.com/jsiebens/ionscale/internal/version"
	"github.com/labstack/echo/v4"
	"net/http"
)

func Version(c echo.Context) error {
	v, r := version.GetReleaseInfo()
	resp := map[string]string{
		"version":  v,
		"revision": r,
	}
	return c.JSON(http.StatusOK, resp)
}
