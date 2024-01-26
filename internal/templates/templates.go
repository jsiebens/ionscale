package templates

import (
	"fmt"
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"io"
)

type Renderer struct {
}

func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if x, ok := data.(templ.Component); ok {
		return layout(x).Render(c.Request().Context(), w)
	}

	return fmt.Errorf("invalid data")
}
