package unit

import "github.com/coreos/go-systemd/dbus"

// DBusConnectionInterface is the interface of a DBus connection
type DBusConnectionInterface interface {
	StartUnit(name string, mode string, ch chan<- string) (int, error)
	StopUnit(name string, mode string, ch chan<- string) (int, error)
	KillUnit(name string, signal int32)
	LinkUnitFiles(files []string, runtime bool, force bool) ([]dbus.LinkUnitFileChange, error)
	Reload() error
}
