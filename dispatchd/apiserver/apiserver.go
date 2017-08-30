package apiserver

import (
	"net"
	"net/http"
	"os"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/template"

	"github.com/innovate-technologies/Dispatch/dispatchd/command"
	"gopkg.in/labstack/echo.v3"
)

type commandInfo struct {
	Command string `json:"command" form:"command" query:"command"`
}

// Config is a pointer need to be set to the main configuration
var Config *config.ConfigurationInfo

// Run starts the HTTP server
func Run() {
	template.Config = Config
	unit.Config = Config

	e := echo.New()
	e.GET("/", getRoot)
	e.GET("/machines", getMachines)

	e.POST("/command", postCommand)

	e.GET("/units", getUnits)
	e.GET("/unit/:name", getUnit)
	e.POST("/unit", postUnit)
	e.POST("/unit/from-template/:template", postUnitFromTemplate)
	e.DELETE("/unit/:name", deleteUnit)
	e.PUT("/unit/:name/start", startUnit)
	e.PUT("/unit/:name/stop", stopUnit)

	e.GET("/templates", getTemplates)
	e.GET("/template/:name", getTemplate)
	e.POST("/template", postTemplate)
	e.DELETE("/template/:name", deleteTemplate)

	os.Remove(Config.BindPath)
	l, err := net.Listen("unix", Config.BindPath)
	if err != nil {
		e.Logger.Fatal(err)
	}
	e.Listener = l
	e.Logger.Fatal(e.Start(""))
}

func getRoot(c echo.Context) error {
	return c.String(http.StatusOK, "Dispatch API server, if you see this on the internet you're doing it wrong")
}

func getMachines(c echo.Context) error {
	return c.String(http.StatusFailedDependency, "I didn't make a machine a struct? oops.... to be continued")
}

func postCommand(c echo.Context) error {
	command.Config = Config
	info := commandInfo{}
	c.Bind(&info)
	commandID := command.SendCommand(info.Command)
	return c.JSON(http.StatusOK, map[string]string{"id": commandID})
}
