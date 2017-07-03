package template

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"text/template"

	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"

	etcd "github.com/coreos/etcd/client"
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
	ctx     = context.Background()
	etcdAPI etcd.KeysAPI
)

// New returns a new blank Template
func New() Template {
	return Template{}
}

// GetAll returns all templates in our zone
func GetAll() ([]Template, error) {
	setUpEtcd()
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/templates/%s", Config.Zone), &etcd.GetOptions{})
	if err != nil {
		return nil, err
	}

	templates := []Template{}

	for _, unitNode := range response.Node.Nodes {
		pathParts := strings.Split(unitNode.Key, "/")
		templates = append(templates, NewFromEtcd(pathParts[len(pathParts)-1]))
	}

	return templates, nil
}

// NewFromEtcd creates a new Template with info from etcd
func NewFromEtcd(name string) Template {
	setUpEtcd()
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
	setUpEtcd()
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
	setUpEtcd()
	etcdAPI.Delete(ctx, fmt.Sprintf("/dispatch/templates/%s/%s", Config.Zone, t.Name), &etcd.DeleteOptions{Recursive: true})
}

// NewUnit created a new unit from the template
func (t *Template) NewUnit(name string, vars map[string]string) {
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

	u.SaveOnEtcd()
	u.PutOnQueue()
}

func setUpEtcd() {
	if etcdAPI != nil {
		return
	}
	c, err := etcd.New(etcd.Config{
		Endpoints:               []string{Config.EtcdAddress},
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: 10 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	etcdAPI = etcd.NewKeysAPI(c)
}

func getKeyFromEtcd(unit, key string) string {
	response, err := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, unit, key), &etcd.GetOptions{})
	if err != nil {
		return ""
	}
	return response.Node.Value
}

func setKeyOnEtcd(template, key, content string) {
	fmt.Println(template, key, content)
	etcdAPI.Set(ctx, fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, template, key), content, &etcd.SetOptions{})
}
