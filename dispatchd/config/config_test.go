package config

import (
	"os"
	"reflect"
	"runtime"
	"testing"
)

func Test_newConfigurationInfo(t *testing.T) {
	hostname, _ := os.Hostname()
	tests := []struct {
		name string
		want ConfigurationInfo
	}{
		{
			name: "default values test",
			want: ConfigurationInfo{
				MachineName: hostname,
				BindIP:      "127.0.0.1",
				BindPort:    7384,
				EtcdAddress: "http://127.0.0.1:2379",
				Zone:        "dc",
				Arch:        runtime.GOARCH,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newConfigurationInfo(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newConfigurationInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readEnv(t *testing.T) {
	type args struct {
		conf *ConfigurationInfo
	}
	tests := []struct {
		name    string
		args    args
		envName string
		envVal  string
		want    *ConfigurationInfo
	}{
		{
			name:    "test machine name",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_MACHINENAME",
			envVal:  "unit",
			want:    &ConfigurationInfo{MachineName: "unit"},
		},
		{
			name:    "test bind ip",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_BINDIP",
			envVal:  "127.0.1.1",
			want:    &ConfigurationInfo{BindIP: "127.0.1.1"},
		},
		{
			name:    "test bind port",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_BINDPORT",
			envVal:  "1234",
			want:    &ConfigurationInfo{BindPort: 1234},
		},
		{
			name:    "test etcd arrd",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_ETCDADDRESS",
			envVal:  "google.com",
			want:    &ConfigurationInfo{EtcdAddress: "google.com"},
		},
		{
			name:    "test public ip",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_PUBLICIP",
			envVal:  "10.0.0.1",
			want:    &ConfigurationInfo{PublicIP: "10.0.0.1"},
		},
		{
			name:    "test zone",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_ZONE",
			envVal:  "PARIS",
			want:    &ConfigurationInfo{Zone: "PARIS"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envName, tt.envVal)
			if readEnv(tt.args.conf); !reflect.DeepEqual(tt.args.conf, tt.want) {
				t.Errorf("readEnv() = %v, want %v", tt.args.conf, tt.want)
			}
			os.Unsetenv(tt.envName)
		})
	}
}
