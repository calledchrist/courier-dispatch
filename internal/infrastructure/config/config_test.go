package config

import (
	"os"
	"testing"
)

func TestLoadConfigurationPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		environment map[string]string
		want        Config
	}{
		{
			name: "defaults without dotenv",
			want: Config{
				App:  AppConfig{Environment: "development"},
				HTTP: HTTPConfig{Host: "0.0.0.0", Port: "8080"},
			},
		},
		{
			name: "prefixed dotenv values",
			file: "COURIER_APP_ENVIRONMENT=staging\nCOURIER_HTTP_HOST=127.0.0.1\nCOURIER_HTTP_PORT=8000\n",
			want: Config{
				App:  AppConfig{Environment: "staging"},
				HTTP: HTTPConfig{Host: "127.0.0.1", Port: "8000"},
			},
		},
		{
			name: "process environment overrides dotenv",
			file: "COURIER_APP_ENVIRONMENT=staging\nCOURIER_HTTP_HOST=127.0.0.1\nCOURIER_HTTP_PORT=8000\n",
			environment: map[string]string{
				"COURIER_APP_ENVIRONMENT": "production",
				"COURIER_HTTP_HOST":       "0.0.0.0",
				"COURIER_HTTP_PORT":       "9000",
			},
			want: Config{
				App:  AppConfig{Environment: "production"},
				HTTP: HTTPConfig{Host: "0.0.0.0", Port: "9000"},
			},
		},
		{
			name:        "process environment without dotenv",
			environment: map[string]string{"COURIER_HTTP_PORT": "9001"},
			want: Config{
				App:  AppConfig{Environment: "development"},
				HTTP: HTTPConfig{Host: "0.0.0.0", Port: "9001"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			for _, key := range []string{
				"COURIER_APP_ENVIRONMENT",
				"COURIER_HTTP_HOST",
				"COURIER_HTTP_PORT",
			} {
				t.Setenv(key, test.environment[key])
			}

			if test.file != "" {
				if err := os.WriteFile(".env", []byte(test.file), 0600); err != nil {
					t.Fatal(err)
				}
			}

			got, err := Load()
			if err != nil {
				t.Fatal(err)
			}

			if *got != test.want {
				t.Fatalf("Load() = %+v, want %+v", *got, test.want)
			}
		})
	}
}

func TestLoadReportsInvalidDotenv(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(".env", []byte("COURIER_HTTP_PORT=\"unterminated\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("invalid dotenv file should return an error")
	}
}
