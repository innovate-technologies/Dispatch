package apiserver

import (
	"net/http"
	"strings"

	"github.com/innovate-technologies/Dispatch/dispatchd/unit/template"
	"gopkg.in/labstack/echo.v3"
)

type templateUnitOptions struct {
	Name  string            `json:"name" form:"name" query:"name"`
	Vars  map[string]string `json:"vars" form:"vars" query:"vars"`
	Ports []int64           `json:"ports" form:"ports" query:"ports"`
}

func getTemplates(c echo.Context) error {
	template := template.NewFromEtcd(c.Param("name"))
	if template.Name == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "error": "Template not found"})
	}
	return c.JSON(http.StatusOK, template)
}

func getTemplate(c echo.Context) error {
	templates, err := template.GetAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"status": "error", "error": err.Error()})
	}
	return c.JSON(http.StatusOK, templates)
}

func postTemplate(c echo.Context) error {
	t := template.New()

	c.Bind(&t) // bind JSON to the unit

	if !strings.Contains(t.Name, "*") {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "Name needs to contain a wildcard (*)"})
	}

	if t.Name == "" || t.UnitContent == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing parameters"})
	}
	// Check if exists
	templateWithName := template.NewFromEtcd(t.Name)
	if templateWithName.Name != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "template already exists"})
	}

	t.SaveOnEtcd()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func deleteTemplate(c echo.Context) error {
	templateWithName := template.NewFromEtcd(c.Param("name"))
	if templateWithName.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "template doesn't exist"})
	}
	templateWithName.Delete()
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func postUnitFromTemplate(c echo.Context) error {
	if c.Param("template") == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing template name"})
	}

	info := templateUnitOptions{}
	c.Bind(&info)

	if info.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "missing unit name"})
	}

	t := template.NewFromEtcd(c.Param("template"))

	if t.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"status": "error", "error": "template doesn't exist"})
	}

	if info.Vars == nil {
		info.Vars = map[string]string{}
	}

	u := t.NewUnit(info.Name, info.Vars)

	// Allow to override ports
	if info.Ports != nil && len(info.Ports) > 0 {
		u.Ports = info.Ports
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
