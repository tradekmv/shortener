package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name              string
		envServerAddress  string
		envBaseURL        string
		wantServerAddress string
		wantBaseURL       string
	}{
		{
			name:              "default values",
			wantServerAddress: "localhost:8080",
			wantBaseURL:       "http://localhost:8080",
		},
		{
			name:              "env variables override defaults",
			envServerAddress:  ":9090",
			envBaseURL:        "https://u.short",
			wantServerAddress: ":9090",
			wantBaseURL:       "https://u.short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SERVER_ADDRESS", tt.envServerAddress)
			os.Setenv("BASE_URL", tt.envBaseURL)
			defer func() {
				os.Unsetenv("SERVER_ADDRESS")
				os.Unsetenv("BASE_URL")
			}()

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.ServerAddress != tt.wantServerAddress {
				t.Errorf("ServerAddress = %v, want %v", cfg.ServerAddress, tt.wantServerAddress)
			}
			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, tt.wantBaseURL)
			}
		})
	}
}
