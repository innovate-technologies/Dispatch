package template

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"text/template"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/etcdclient"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"

	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

// Template contains all info of a template
type Template struct {
	Name          string            `json:"name"`
	Ports         []int64           `json:"ports"`
	Constraints   map[string]string `json:"constraints"`
	UnitContent   string            `json:"unitContent"`
	MaxPerMachine int64             `json:"maxPerMachine"`
	onEtcd        bool
}

var (
	// Config is a pointer need to be set to the main configuration
	Config  *config.ConfigurationInfo
	ctx             = context.Background()
	etcdAPI etcd.KV = etcdclient.GetEtcdv3()
)

// New returns a new blank Template
func New() Template {
	return Template{}
}

// GetAll returns all templates in our zone
func GetAll() ([]Template, error) {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/templates", Config.Zone), etcd.WithPrefix())
	if err != nil {
		return nil, err
	}

	templates := []Template{}
	hadTemplateNames := map[string]bool{}

	for _, kv := range response.Kvs {
		pathParts := strings.Split(string(kv.Key), "/")
		if _, ok := hadTemplateNames[pathParts[4]]; !ok {
			templates = append(templates, NewFromEtcd(pathParts[4]))
			hadTemplateNames[pathParts[4]] = true
		}

	}

	return templates, nil
}

// NewFromEtcd creates a new Template with info from etcd
func NewFromEtcd(name string) Template {
	template := New()
	template.onEtcd = true
	template.Name = getKeyFromEtcd(name, "name")
	template.UnitContent = getKeyFromEtcd(name, "unit")
	template.MaxPerMachine, _ = strconv.ParseInt(getKeyFromEtcd(name, "maxpermachine"), 10, 64)

	template.Ports = []int64{}
	portsStringArray := strings.Split(getKeyFromEtcd(name, "ports"), ",")
	for _, portString := range portsStringArray {
		port, err := strconv.ParseInt(portString, 10, 64)
		if err == nil {
			template.Ports = append(template.Ports, port)
		}
	}

	return template
}

// SaveOnEtcd saves the unit to etcd
func (t *Template) SaveOnEtcd() {
	setKeyOnEtcd(t.Name, "name", t.Name)
	setKeyOnEtcd(t.Name, "unit", t.UnitContent)
	setKeyOnEtcd(t.Name, "maxpermachine", strconv.FormatInt(t.MaxPerMachine, 10))

	portStrings := []string{}
	for _, port := range t.Ports {
		portStrings = append(portStrings, strconv.FormatInt(port, 10))
	}

	setKeyOnEtcd(t.Name, "ports", strings.Join(portStrings, ","))
	t.onEtcd = true
}

// Delete removes the template from etcd
func (t *Template) Delete() {
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/%s/templates/%s", Config.Zone, t.Name), etcd.WithPrefix())
}

// NewUnit gives back a new Unit from the template
func (t *Template) NewUnit(name string, vars map[string]string) unit.Unit {
	u := unit.New()
	u.Name = strings.Replace(t.Name, "*", name, -1)
	u.Template = t.Name
	u.DesiredState = state.Active
	u.Ports = t.Ports

	// parse unit content
	var unit bytes.Buffer
	unitTemplate := template.New("unit template")
	unitTemplate, _ = unitTemplate.Parse(t.UnitContent)
	vars["name"] = name
	unitTemplate.Execute(&unit, vars)
	u.UnitContent = unit.String()

	return u
}

func getKeyFromEtcd(unit, key string) string {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/templates/%s/%s", Config.Zone, unit, key))
	if err != nil || response.Count == 0 {
		return ""
	}
	return string(response.Kvs[0].Value)
}

func setKeyOnEtcd(unit, key, content string) {
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/templates/%s/%s", Config.Zone, unit, key), content)
}
