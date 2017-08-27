package unit

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	etcd "github.com/coreos/etcd/client"
	"github.com/golang/mock/gomock"
	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"
	"github.com/innovate-technologies/Dispatch/mocks/dbusmock"
	etcdMock "github.com/innovate-technologies/Dispatch/mocks/etcdmock"
	"github.com/spf13/afero"
)

func setUpMockEtcd(t *testing.T) (*etcdMock.MockKeysAPI, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockAPI := etcdMock.NewMockKeysAPI(ctrl)

	EtcdAPI = mockAPI

	return mockAPI, ctrl
}

func setUpMockDBus(t *testing.T) (*dbusmock.MockDBusConnectionInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockDBus := dbusmock.NewMockDBusConnectionInterface(gomock.NewController(t))

	DBusConnection = mockDBus

	return mockDBus, ctrl
}

func setUpConfig() {
	Config = &config.ConfigurationInfo{Zone: "test"}
}

func getTestUnit() Unit {
	return Unit{
		Name:         "test-unit.service",
		Machine:      "test-machine",
		Template:     "",
		Global:       "",
		State:        state.Dead,
		DesiredState: state.Dead,
		Ports:        []int64{80, 443},
		UnitContent:  "TEST CONTENT",
	}
}

func Test_New(t *testing.T) {
	setUpMockDBus(t)
	tests := []struct {
		name string
		want Unit
	}{
		{
			name: "Default",
			want: Unit{},
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

func Test_newFromEtcd(t *testing.T) {
	setUpConfig()
	setUpMockDBus(t)
	mockEtcd, ctrl := setUpMockEtcd(t)
	defer ctrl.Finish()

	unitName := "test-unit.service"

	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "name"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: unitName}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "machine"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "test-machine"}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "global"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: ""}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "unit"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "test content"}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "desiredState"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "active"}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "template"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: ""}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "ports"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "80,443"}}, nil)
	mockEtcd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "state"), gomock.Any()).Return(&etcd.Response{Node: &etcd.Node{Value: "active"}}, nil)

	want := Unit{
		Name:         unitName,
		Machine:      "test-machine",
		Global:       "",
		UnitContent:  "test content",
		State:        state.Active,
		DesiredState: state.Active,
		Template:     "",
		Ports:        []int64{80, 443},
		onEtcd:       true,
		etcdName:     unitName,
	}

	if got := NewFromEtcd("test-unit"); !reflect.DeepEqual(got, want) {
		t.Errorf("Got %v, want %v", got, want)
	}
}

func Test_start(t *testing.T) {
	setUpConfig()
	mockDBus, ctrl := setUpMockDBus(t)
	mockEtcd, ctrl2 := setUpMockEtcd(t)
	defer ctrl.Finish()
	defer ctrl2.Finish()

	unit := getTestUnit()

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "starting", gomock.Any())

	mockDBus.EXPECT().StartUnit(unit.Name, "fail", gomock.Any())

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "active", gomock.Any())

	unit.Start()
}

func Test_stop(t *testing.T) {
	setUpConfig()
	mockDBus, ctrl := setUpMockDBus(t)
	mockEtcd, ctrl2 := setUpMockEtcd(t)
	defer ctrl.Finish()
	defer ctrl2.Finish()

	unit := getTestUnit()

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "dead", gomock.Any())

	unit.Stop()
}

func Test_stopError(t *testing.T) {
	setUpConfig()
	mockDBus, ctrl := setUpMockDBus(t)
	mockEtcd, ctrl2 := setUpMockEtcd(t)
	defer ctrl.Finish()
	defer ctrl2.Finish()

	unit := getTestUnit()

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any()).Return(0, fmt.Errorf("test"))
	mockDBus.EXPECT().KillUnit(gomock.Any(), gomock.Any())

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "dead", gomock.Any())

	unit.Stop()
}

func Test_restart(t *testing.T) {
	setUpConfig()
	mockDBus, ctrl := setUpMockDBus(t)
	mockEtcd, ctrl2 := setUpMockEtcd(t)
	defer ctrl.Finish()
	defer ctrl2.Finish()

	unit := getTestUnit()

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "dead", gomock.Any())

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "starting", gomock.Any())

	mockDBus.EXPECT().StartUnit(unit.Name, "fail", gomock.Any())

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "active", gomock.Any())

	unit.Restart()
}

func Test_create(t *testing.T) {
	setUpConfig()
	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()
	FS = afero.NewMemMapFs()

	unit := getTestUnit()

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any()).Return(0, nil)
	mockDBus.EXPECT().LinkUnitFiles([]string{unitPath + unit.Name}, true, true)
	mockDBus.EXPECT().Reload()

	unit.Create()

	if !unit.onDisk {
		t.Errorf("Unit is not set as on disk")
	}
	_, err := FS.Stat(unitPath + unit.Name)
	if os.IsNotExist(err) {
		t.Errorf("file does not exist.\n")
	}
}

func Test_putOnQueue(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	defer ctrl.Finish()

	unit := getTestUnit()

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), unit.Name, gomock.Any())

	unit.PutOnQueue()
}

func Test_putOnQueueGlobal(t *testing.T) {
	setUpConfig()
	unit := getTestUnit()
	unit.Global = unit.Name

	unit.PutOnQueue()
}

func Test_saveOnEtcd(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	defer ctrl.Finish()

	unit := getTestUnit()

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "name"), unit.Name, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "unit"), unit.UnitContent, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "template"), unit.Template, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "desiredState"), "dead", gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "ports"), "80,443", gomock.Any())

	unit.SaveOnEtcd()

	if !unit.onEtcd {
		t.Errorf("Unit is not set as on etcd")
	}
}

func Test_saveOnEtcdWithGlobal(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	defer ctrl.Finish()

	unit := getTestUnit()
	unit.Global = unit.Name

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "name"), unit.Name, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "unit"), unit.UnitContent, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "template"), unit.Template, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "desiredState"), "dead", gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "ports"), "80,443", gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "global"), unit.Global, gomock.Any())
	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), unit.Name, gomock.Any())

	unit.SaveOnEtcd()

	if !unit.onEtcd {
		t.Errorf("Unit is not set as on etcd")
	}
}

func Test_destroy(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	mockDBus, ctrl2 := setUpMockDBus(t)
	defer ctrl.Finish()
	defer ctrl2.Finish()
	FS = afero.NewMemMapFs()

	unit := getTestUnit()
	unit.onDisk = true
	unit.onEtcd = true

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())
	mockDBus.EXPECT().Reload()

	mockEtcd.EXPECT().Set(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "dead", gomock.Any())

	mockEtcd.EXPECT().Delete(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s", Config.Zone, unit.Name), &etcd.DeleteOptions{Recursive: true})
	mockEtcd.EXPECT().Delete(gomock.Any(), fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, Config.MachineName, unit.Name), &etcd.DeleteOptions{Recursive: true})

	unit.Destroy()

	if unit.onDisk {
		t.Errorf("Unit is not unset as on disk")
	}
	if unit.onEtcd {
		t.Errorf("Unit is not unset as on etcd")
	}
}

func Test_destroyGlobal(t *testing.T) {
	setUpConfig()
	mockEtcd, ctrl := setUpMockEtcd(t)
	mockDBus, ctrl2 := setUpMockDBus(t)
	defer ctrl.Finish()
	defer ctrl2.Finish()
	FS = afero.NewMemMapFs()

	unit := getTestUnit()
	unit.onDisk = true
	unit.onEtcd = true
	unit.Global = unit.Name

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())
	mockDBus.EXPECT().Reload()

	mockEtcd.EXPECT().Delete(gomock.Any(), fmt.Sprintf("/dispatch/%s/units/%s", Config.Zone, unit.Name), &etcd.DeleteOptions{Recursive: true})
	mockEtcd.EXPECT().Delete(gomock.Any(), fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, Config.MachineName, unit.Name), &etcd.DeleteOptions{Recursive: true})
	mockEtcd.EXPECT().Delete(gomock.Any(), fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), gomock.Any())

	unit.Destroy()

	if unit.onDisk {
		t.Errorf("Unit is not unset as on disk")
	}
	if unit.onEtcd {
		t.Errorf("Unit is not unset as on etcd")
	}
}

func Test_killAllOldUnitsWithNone(t *testing.T) {
	setUpConfig()
	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()
	FS = afero.NewMemMapFs()

	mockDBus.EXPECT().Reload()

	KillAllOldUnits()

	_, err := FS.Stat(unitPath)
	if os.IsNotExist(err) {
		t.Errorf("Unit directory does not exist.\n")
	}
}

func Test_killAllOldUnits(t *testing.T) {
	setUpConfig()
	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()

	FS = afero.NewMemMapFs()
	FS.Create(unitPath + "test1.service")
	FS.Create(unitPath + "test2.service")

	mockDBus.EXPECT().KillUnit("test1.service", gomock.Any())
	mockDBus.EXPECT().KillUnit("test2.service", gomock.Any())
	mockDBus.EXPECT().Reload()

	KillAllOldUnits()

	_, err := FS.Stat(unitPath + "test1.service")
	if !os.IsNotExist(err) {
		t.Errorf("file still exists.\n")
	}
}
