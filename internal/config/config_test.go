package config_test

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ya-lobur/go-url-shortener/internal/config"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want config.Config
	}{
		{
			name: "defaults",
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8080"},
				Handler: config.HandlerConfig{BaseURL: "http://localhost:8080"},
			},
		},
		{
			name: "address_flag",
			args: []string{"-a", "localhost:8081"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8081"},
				Handler: config.HandlerConfig{BaseURL: "http://localhost:8080"},
			},
		},
		{
			name: "base_url_flag",
			args: []string{"-b", "https://flags.example"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8080"},
				Handler: config.HandlerConfig{BaseURL: "https://flags.example"},
			},
		},
		{
			name: "both_flags",
			args: []string{"-a", "localhost:8081", "-b", "https://flags.example"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8081"},
				Handler: config.HandlerConfig{BaseURL: "https://flags.example"},
			},
		},
		{
			name: "address_environment",
			env:  map[string]string{"SERVER_ADDRESS": "localhost:8082"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8082"},
				Handler: config.HandlerConfig{BaseURL: "http://localhost:8080"},
			},
		},
		{
			name: "base_url_environment",
			env:  map[string]string{"BASE_URL": "https://env.example"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8080"},
				Handler: config.HandlerConfig{BaseURL: "https://env.example"},
			},
		},
		{
			name: "both_environment_variables",
			env:  map[string]string{"SERVER_ADDRESS": "localhost:8082", "BASE_URL": "https://env.example"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8082"},
				Handler: config.HandlerConfig{BaseURL: "https://env.example"},
			},
		},
		{
			name: "environment_overrides_flags",
			args: []string{"-a", "localhost:8081", "-b", "https://flags.example"},
			env:  map[string]string{"SERVER_ADDRESS": "localhost:8082", "BASE_URL": "https://env.example"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8082"},
				Handler: config.HandlerConfig{BaseURL: "https://env.example"},
			},
		},
		{
			name: "environment_address_and_flag_base_url",
			args: []string{"-b", "https://flags.example"},
			env:  map[string]string{"SERVER_ADDRESS": "localhost:8082"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8082"},
				Handler: config.HandlerConfig{BaseURL: "https://flags.example"},
			},
		},
		{
			name: "flag_address_and_environment_base_url",
			args: []string{"-a", "localhost:8081"},
			env:  map[string]string{"BASE_URL": "https://env.example"},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8081"},
				Handler: config.HandlerConfig{BaseURL: "https://env.example"},
			},
		},
		{
			name: "empty_environment_keeps_flags",
			args: []string{"-a", "localhost:8081", "-b", "https://flags.example"},
			env:  map[string]string{"SERVER_ADDRESS": "", "BASE_URL": ""},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8081"},
				Handler: config.HandlerConfig{BaseURL: "https://flags.example"},
			},
		},
		{
			name: "empty_environment_keeps_defaults",
			env:  map[string]string{"SERVER_ADDRESS": "", "BASE_URL": ""},
			want: config.Config{
				HTTP:    config.HTTPConfig{RunAddr: "localhost:8080"},
				Handler: config.HandlerConfig{BaseURL: "http://localhost:8080"},
			},
		},
	}

	// These subtests change process-wide state and must run sequentially.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, name := range []string{"SERVER_ADDRESS", "BASE_URL"} {
				// Register restoration before removing the variable entirely.
				t.Setenv(name, "")
				require.NoError(t, os.Unsetenv(name))
			}
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			originalArgs, originalFlags := os.Args, flag.CommandLine
			t.Cleanup(func() {
				os.Args, flag.CommandLine = originalArgs, originalFlags
			})
			os.Args = append([]string{"shortener"}, tt.args...)
			flag.CommandLine = flag.NewFlagSet("shortener", flag.ExitOnError)

			assert.Equal(t, tt.want, config.Parse())
		})
	}
}
