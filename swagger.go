package main

// module deps
import "io"
import "fmt"
import "net/url"
import "net/http"
import "html/template"
import "github.com/labstack/echo"

// Template for swagger yaml
type Template struct {
	templates *template.Template
}

var localHost = "127.0.0.1"

// Render swagger yaml basepath (host:port)
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// serve swagger docs
func swagger(ctx echo.Context) error {
	if *bindAddress == "" {
		*bindAddress = localHost
	}

	swaggerSvc := "http://petstore.swagger.io"
	srvaddr := fmt.Sprintf("http://%s:%s/swagger.yaml", *bindAddress, *bindPort)
	swaggerLocation := fmt.Sprintf("%s?url=%s", swaggerSvc, url.PathEscape(srvaddr))
	return ctx.Redirect(http.StatusMovedPermanently, swaggerLocation)
}

// render swagger yaml
func renderSwagger(ctx echo.Context) error {
	if *bindAddress == "" {
		*bindAddress = localHost
	}

	data := struct {
		Host string
		Port string
	}{
		*bindAddress,
		*bindPort,
	}

	return ctx.Render(http.StatusOK, "swagger.yaml", data)
}
