package apiserver

import (
	"net/http"

	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	state "github.com/innovate-technologies/Dispatch/dispatchd/unit/state"
	"gopkg.in/labstack/echo.v3"
)

func getUnits(c echo.Context) error {
	units, err := unit.GetAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "error": err.Error()})
	}
	return c.JSON(http.StatusOK, units)
}

func getUnit(c echo.Context) error {
	unit := unit.NewFromEtcd(c.Param("name"))
	if unit.Name == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "error": "Unit does not exist"})
	}
	return c.JSON(http.StatusOK, unit)
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

	if u.Machine == "" && u.Global == "" { // Not running
		go u.Destroy()
	} else {
		go u.SetDesiredState(state.Destroy)
	}

	u.WaitOnDestroy()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func stopUnit(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing name"})
	}
	u := unit.NewFromEtcd(name)
	if u.UnitContent == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "unit does not exist"})
	}

	u.SetDesiredState(state.Dead)
	u.WaitOnState(state.Dead)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func startUnit(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing name"})
	}
	u := unit.NewFromEtcd(name)
	if u.UnitContent == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "unit does not exist"})
	}

	u.SetDesiredState(state.Active)
	u.WaitOnState(state.Active)

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
