package app

import "testing"

func TestLoadConfig(t *testing.T) {
	for _, address := range []string{"127.0.0.1:8080", ":8080", "[::1]:8080", "localhost:9000"} {
		t.Run(address, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", address)
			cfg, err := LoadConfig()
			if err != nil || cfg.HTTPAddr != address {
				t.Fatalf("LoadConfig() = %+v, %v", cfg, err)
			}
		})
	}
	for _, address := range []string{"", "8080", "localhost:http", ":0", ":-1", ":65536", "http://localhost:8080"} {
		t.Run("invalid_"+address, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", address)
			if _, err := LoadConfig(); err == nil {
				t.Fatal("invalid HTTP_ADDR accepted")
			}
		})
	}
}
