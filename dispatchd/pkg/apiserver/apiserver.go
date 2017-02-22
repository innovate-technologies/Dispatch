package apiserver

import (
	"net/http"

	"gopkg.in/labstack/echo.v3"
)

// Run starts the HTTP server
func Run() {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.Logger.Fatal(e.Start(":1323"))
}
