package template

import (
	"fmt"
	"reflect"
	"testing"

	etcd "github.com/coreos/etcd/client"
	"github.com/golang/mock/gomock"
	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"
	"github.com/innovate-technologies/Dispatch/mocks/dbusmock"
	etcdMock "github.com/innovate-technologies/Dispatch/mocks/etcdmock"
)

func setUpMockEtcd(t *testing.T) (*etcdMock.MockKeysAPI, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockAPI := etcdMock.NewMockKeysAPI(ctrl)

	etcdAPI = mockAPI

	return mockAPI, ctrl
}

func setUpConfig() {
	Config = &config.ConfigurationInfo{Zone: "test"}
}

func getTestTemplate() Template {
	template := New()
	template.Name = "test-temp-*"
	template.Ports = []int64{80, 443}
	// TO DO: add constraints
	template.UnitContent = "hello"
	template.MaxPerMachine = 5
	template.onEtcd = false

	return template
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want Template
	}{
		{
			name: "empty new",
			want: Template{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveOnEtcd(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	defer ctrl.Finish()

	template := getTestTemplate()

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, template.Name, "name"), template.Name, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, template.Name, "unit"), template.UnitContent, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, template.Name, "maxpermachine"), "5", gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, template.Name, "ports"), "80,443", gomock.Any())

	template.SaveOnEtcd()
}

func Test_deleteFromEtcd(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	defer ctrl.Finish()

	template := getTestTemplate()

	mockEtcd.EXPECT().Delete(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s", Config.Zone, template.Name), &etcd.DeleteOptions{Recursive: true})

	template.Delete()
}

func Test_newUnit(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	mockDBus := dbusmock.NewMockDBusConnectionInterface(gomock.NewController(t))
	defer ctrl.Finish()

	unit.EtcdAPI = mockEtcd
	unit.DBusConnection = mockDBus
	unit.Config = Config

	template := getTestTemplate()

	unitName := "test-temp-test"

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unitName, "name"), unitName, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unitName, "unit"), template.UnitContent, gomock.Any()) // TO DO: add vars
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unitName, "template"), template.Name, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unitName, "desiredState"), state.Active.String(), gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/units/%s/%s/%s", Config.Zone, unitName, "ports"), "80,443", gomock.Any())

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/queue/%s/%s", Config.Zone, unitName), unitName, gomock.Any())

	template.NewUnit("test")
}

func Test_newFromEtcd(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	defer ctrl.Finish()

	templateName := "test-temp"

	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, templateName, "unit"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "test content"}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, templateName, "maxpermachine"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "10"}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/templates/%s/%s/%s", Config.Zone, templateName, "ports"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "80,443"}}, nil)

	template := NewFromEtcd(templateName)

	if template.Name != templateName {
		t.Errorf("template.Name = %v, want %v", template.Name, templateName)
	}
	if template.MaxPerMachine != 10 {
		t.Errorf("template.NamMaxPerMachinee = %v, want %v", template.MaxPerMachine, 10)
	}
	if template.Ports[0] != 80 {
		t.Errorf("template.Ports[0] = %v, want %v", template.Ports[0], 80)
	}
	if template.Ports[1] != 443 {
		t.Errorf("template.Ports[1] = %v, want %v", template.Ports[1], 443)
	}
}
