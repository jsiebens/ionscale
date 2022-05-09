package templates

import (
	"embed"
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
)

//go:embed *.html
var fs embed.FS

func NewTemplates() *Template {
	return &Template{templates: template.Must(template.ParseFS(fs, "*.html"))}
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
