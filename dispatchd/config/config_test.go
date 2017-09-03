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
				BindPath:    "/var/run/dispatch.socket",
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
			envName: "DISPATCH_BINDPATH",
			envVal:  "/tmp/test.sock",
			want:    &ConfigurationInfo{BindPath: "/tmp/test.sock"},
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
		{
			name:    "test username",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_ETCD_USERNAME",
			envVal:  "root",
			want:    &ConfigurationInfo{EtcdAuth: etcdAuth{Username: "root"}},
		},
		{
			name:    "test password",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_ETCD_PASSWORD",
			envVal:  "rootpass",
			want:    &ConfigurationInfo{EtcdAuth: etcdAuth{Password: "rootpass"}},
		},
		{
			name:    "test password",
			args:    args{conf: &ConfigurationInfo{}},
			envName: "DISPATCH_ETCD_CA",
			envVal:  "/etc/ssl/etcd/ca.pem",
			want:    &ConfigurationInfo{EtcdTLS: etcdTLS{CACert: "/etc/ssl/etcd/ca.pem"}},
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
