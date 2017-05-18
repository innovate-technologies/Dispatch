package apiserver

import (
	"net/http"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	state "github.com/innovate-technologies/Dispatch/dispatchd/unit/state"

	"strconv"

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
	e := echo.New()
	e.GET("/", getRoot)
	e.GET("/machines", getMachines)
	e.GET("/units", getUnits)
	e.POST("/command", postCommand)
	e.POST("/unit", postUnit)
	e.DELETE("/unit/:name", deleteUnit)
	e.Logger.Fatal(e.Start(Config.BindIP + ":" + strconv.Itoa(Config.BindPort)))
}

func getRoot(c echo.Context) error {
	return c.String(http.StatusOK, "Dispatch API server, if you see this on the internet you're doing it wrong")
}

func getMachines(c echo.Context) error {
	return c.String(http.StatusFailedDependency, "I didn't make a machine a struct? oops.... to be continued")
}

func getUnits(c echo.Context) error {
	units, err := unit.GetAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "error": err.Error()})
	}
	return c.JSON(http.StatusOK, units)
}

func postCommand(c echo.Context) error {
	command.Config = Config
	info := commandInfo{}
	c.Bind(&info)
	commandID := command.SendCommand(info.Command)
	return c.JSON(http.StatusOK, map[string]string{"id": commandID})
}

func postUnit(c echo.Context) error {
	u := unit.New()
	u.DesiredState = state.Active

	c.Bind(&u) // bind JSON to the unit
	if u.Name == "" || u.UnitContent == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing parameters"})
	}
	// Check if exists
	unitWithName := unit.NewFromEtcd(u.Name)
	if unitWithName.Name != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "unit already exists"})
	}

	u.SaveOnEtcd()
	u.PutOnQueue()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func deleteUnit(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing name"})
	}
	u := unit.NewFromEtcd(name)
	if u.UnitContent == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "unit does not exist"})
	}
	u.SetDesiredState(state.Destroy)
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
