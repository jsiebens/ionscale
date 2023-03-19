package handlers

import (
	"github.com/jsiebens/ionscale/internal/version"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
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

func logError(err error) error {
	zap.L().WithOptions(zap.AddCallerSkip(1)).Error("error processing request", zap.Error(err))
	return err
}
