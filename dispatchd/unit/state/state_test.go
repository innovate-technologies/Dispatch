package state

import (
	"reflect"
	"testing"
)

func TestState_String(t *testing.T) {
	tests := []struct {
		name string
		s    State
		want string
	}{
		{
			name: "active",
			s:    Active,
			want: "active",
		},
		{
			name: "dead",
			s:    Dead,
			want: "dead",
		},
		{
			name: "starting",
			s:    Starting,
			want: "starting",
		},
		{
			name: "destroy",
			s:    Destroy,
			want: "destroy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("State.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForString(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want State
	}{
		{
			name: "no match",
			args: args{name: "lolcat"},
			want: Dead,
		},
		{
			name: "active",
			args: args{name: "active"},
			want: Active,
		},
		{
			name: "dead",
			args: args{name: "dead"},
			want: Dead,
		},
		{
			name: "starting",
			args: args{name: "starting"},
			want: Starting,
		},
		{
			name: "destroy",
			args: args{name: "destroy"},
			want: Destroy,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ForString(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ForString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForInt(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name string
		args args
		want State
	}{
		{
			name: "active",
			args: args{i: 0},
			want: Active,
		},
		{
			name: "dead",
			args: args{i: 1},
			want: Dead,
		},
		{
			name: "starting",
			args: args{i: 2},
			want: Starting,
		},
		{
			name: "destroy",
			args: args{i: 3},
			want: Destroy,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ForInt(tt.args.i); got != tt.want {
				t.Errorf("ForInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
