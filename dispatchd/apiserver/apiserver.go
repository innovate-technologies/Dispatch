package apiserver

import (
	"net/http"

	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	state "github.com/innovate-technologies/Dispatch/dispatchd/unit/state"

	"gopkg.in/labstack/echo.v3"
)

// Run starts the HTTP server
func Run() {
	e := echo.New()
	e.GET("/", getRoot)
	e.GET("/:zone/machines", getMachines)
	e.GET("/:zone/units", getUnits)
	e.PUT("/:zone/command", putCommand)
	e.PUT("/:zone/unit", putUnit)
	e.DELETE("/:zone/unit/:name", deleteUnit)
	e.Logger.Fatal(e.Start(":1323"))
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

func putCommand(c echo.Context) error {
	return c.String(http.StatusTeapot, "Heh, sorry had no time... //TO DO")
}

func putUnit(c echo.Context) error {
	u := unit.New()
	c.Bind(&u) // bind JSON to the unit
	if u.Name == "" || u.UnitContent == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing parameters"})
	}
	u.DesiredState = state.Active
	u.SaveOnEtcd()
	u.PutOnQueue()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func deleteUnit(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing name"})
	}
	u := unit.NewFromEtcd(name)
	if u.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "unit does not exist"})
	}
	u.Destroy()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
