package unit

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/innovate-technologies/Dispatch/dispatchd/config"
	"github.com/innovate-technologies/Dispatch/dispatchd/unit/state"
	"github.com/innovate-technologies/Dispatch/mocks/dbusmock"
	"github.com/innovate-technologies/Dispatch/test/etcdserver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func setUpMockDBus(t *testing.T) (*dbusmock.MockDBusConnectionInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockDBus := dbusmock.NewMockDBusConnectionInterface(gomock.NewController(t))

	DBusConnection = mockDBus

	return mockDBus, ctrl
}

func init() {
	Config = &config.ConfigurationInfo{Zone: "test", MachineName: "test-machine"}
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

func assertEtcd(t *testing.T, key, result string) {
	res, _ := etcdAPI.Get(ctx, key)
	if res.Count == 0 {
		t.Fail()
		return
	}
	assert.Equal(t, result, string(res.Kvs[0].Value))
}

func assertEmptyEtcd(t *testing.T, key string) {
	res, _ := etcdAPI.Get(ctx, key)
	if res.Count != 0 {
		t.Fail()
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
	etcdserver.Start()
	defer etcdserver.Stop()

	_, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()

	unitName := "test-unit.service"

	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "name"), unitName)
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "machine"), "test-machine")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "global"), "")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "unit"), "test content")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "desiredState"), "active")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "template"), "")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "ports"), "80,443")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unitName, "state"), "active")

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
	etcdserver.Start()
	defer etcdserver.Stop()

	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()

	unit := getTestUnit()

	mockDBus.EXPECT().StartUnit(unit.Name, "fail", gomock.Any())

	unit.Start()
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "active")
}

func Test_stop(t *testing.T) {
	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()

	unit := getTestUnit()

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())

	unit.Stop()
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "dead")
}

func Test_stopError(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()

	unit := getTestUnit()

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any()).Return(0, fmt.Errorf("test"))
	mockDBus.EXPECT().KillUnit(gomock.Any(), gomock.Any())

	unit.Stop()
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "dead")
}

func Test_restart(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()

	unit := getTestUnit()

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())

	unit.Restart()
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/state", Config.Zone, unit.Name), "active")
}

func Test_create(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()
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
	etcdserver.Start()
	defer etcdserver.Stop()

	unit := getTestUnit()
	unit.PutOnQueue()
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), unit.Name)
}

func Test_putOnQueueGlobal(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

	unit := getTestUnit()
	unit.Global = unit.Name

	unit.PutOnQueue()
	kv, _ := etcdAPI.Get(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name))
	assert.Equal(t, int64(0), kv.Count)
}

func Test_saveOnEtcd(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

	unit := getTestUnit()
	unit.SaveOnEtcd()

	if !unit.onEtcd {
		t.Errorf("Unit is not set as on etcd")
	}
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "name"), unit.Name)
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "unit"), unit.UnitContent)
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "template"), unit.Template)
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "desiredState"), "dead")
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "ports"), "80,443")
}

func Test_saveOnEtcdWithGlobal(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

	unit := getTestUnit()
	unit.Global = unit.Name

	unit.SaveOnEtcd()

	if !unit.onEtcd {
		t.Errorf("Unit is not set as on etcd")
	}
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "name"), unit.Name)
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "unit"), unit.UnitContent)
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "template"), unit.Template)
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "desiredState"), "dead")
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/%s", Config.Zone, unit.Name, "ports"), "80,443")
	assertEtcd(t, fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), unit.Name)
}

func Test_destroy(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()

	FS = afero.NewMemMapFs()

	unit := getTestUnit()
	unit.onDisk = true
	unit.onEtcd = true

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())
	mockDBus.EXPECT().Reload()

	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), "test")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/name", Config.Zone, unit.Name), "test")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, Config.MachineName, unit.Name), "test")

	unit.Destroy()

	if unit.onDisk {
		t.Errorf("Unit is not unset as on disk")
	}
	if unit.onEtcd {
		t.Errorf("Unit is not unset as on etcd")
	}
	assertEmptyEtcd(t, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name))
	assertEmptyEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/name", Config.Zone, unit.Name))
	assertEmptyEtcd(t, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, Config.MachineName, unit.Name))
}

func Test_destroyGlobal(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

	mockDBus, ctrl := setUpMockDBus(t)
	defer ctrl.Finish()
	FS = afero.NewMemMapFs()

	unit := getTestUnit()
	unit.onDisk = true
	unit.onEtcd = true
	unit.Global = unit.Name

	mockDBus.EXPECT().StopUnit(unit.Name, "fail", gomock.Any())
	mockDBus.EXPECT().Reload()

	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name), "test")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/units/%s/name", Config.Zone, unit.Name), "test")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, Config.MachineName, unit.Name), "test")
	etcdAPI.Put(ctx, fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name), "test")

	unit.Destroy()

	if unit.onDisk {
		t.Errorf("Unit is not unset as on disk")
	}
	if unit.onEtcd {
		t.Errorf("Unit is not unset as on etcd")
	}

	assertEmptyEtcd(t, fmt.Sprintf("/dispatch/%s/queue/%s", Config.Zone, unit.Name))
	assertEmptyEtcd(t, fmt.Sprintf("/dispatch/%s/units/%s/name", Config.Zone, unit.Name))
	assertEmptyEtcd(t, fmt.Sprintf("/dispatch/%s/machines/%s/units/%s", Config.Zone, Config.MachineName, unit.Name))
	assertEmptyEtcd(t, fmt.Sprintf("/dispatch/%s/globals/%s", Config.Zone, unit.Name))
}

func Test_killAllOldUnitsWithNone(t *testing.T) {
	etcdserver.Start()
	defer etcdserver.Stop()

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
	etcdserver.Start()
	defer etcdserver.Stop()

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
