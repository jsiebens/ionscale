package handlers

import (
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"tailscale.com/tailcfg"
)

const (
	SupportedCapabilityVersion      = 68
	NoiseCapabilityVersion          = 28
	UnsupportedClientVersionMessage = "ionscale only support client version >= 1.48.0, please upgrade your client"
)

var UnsupportedClientVersionError = echo.NewHTTPError(http.StatusBadRequest, UnsupportedClientVersionMessage)

func KeyHandler(keys *config.ServerKeys) echo.HandlerFunc {
	legacyPublicKey := keys.LegacyControlKey.Public()
	publicKey := keys.ControlKey.Public()

	return func(c echo.Context) error {
		v := c.QueryParam("v")

		if v != "" {
			clientCapabilityVersion, err := strconv.Atoi(v)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid version")
			}

			if clientCapabilityVersion >= NoiseCapabilityVersion {
				resp := tailcfg.OverTLSPublicKeyResponse{
					LegacyPublicKey: legacyPublicKey,
					PublicKey:       publicKey,
				}
				return c.JSON(http.StatusOK, resp)
			}
		}

		return c.String(http.StatusOK, legacyPublicKey.UntypedHexString())
	}
}
