package mongo

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/douyu/jupiter/pkg/conf"
)

func TestStdConfig(t *testing.T) {
	sr := strings.NewReader("[jupiter.mongo.demo]\nsocketTimeout=\"5s\"\npoolLimit=100")
	if err := conf.LoadFromReader(sr, toml.Unmarshal); err != nil {
		panic(err)
	}

	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want Config
	}{
		// TODO: Add test cases.
		{
			name: "std config",
			args: args{
				name: "demo",
			},
			want: Config{
				DSN:           "",
				SocketTimeout: time.Second * 5,
				PoolLimit:     100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StdConfig(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StdConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRawConfig(t *testing.T) {
	sr := strings.NewReader("[minerva.mongo.demo]\nsocketTimeout=\"5s\"\npoolLimit=100")
	if err := conf.LoadFromReader(sr, toml.Unmarshal); err != nil {
		panic(err)
	}

	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want Config
	}{
		// TODO: Add test cases.
		{
			name: "raw config",
			args: args{
				key: "minerva.mongo.demo",
			},
			want: Config{
				DSN:           "",
				SocketTimeout: time.Second * 5,
				PoolLimit:     100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RawConfig(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RawConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
