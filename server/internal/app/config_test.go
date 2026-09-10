package app

import (
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
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

func TestLoadConfigRequiresDatabaseURL(t *testing.T) {
	for _, value := range []string{"", "http://127.0.0.1/formtally", "postgres://127.0.0.1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DATABASE_URL", value)
			if _, err := LoadConfig(); err == nil {
				t.Fatalf("invalid DATABASE_URL %q accepted", value)
			}
		})
	}
}

func TestLoadConfigParsesAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, http://127.0.0.1:5173")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://app.example.com", "http://127.0.0.1:5173"}
	if !reflect.DeepEqual(cfg.AllowedOrigins, want) {
		t.Fatalf("AllowedOrigins = %#v, want %#v", cfg.AllowedOrigins, want)
	}
}

func TestLoadConfigRejectsInvalidAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	for _, value := range []string{"", "app.example.com", "ftp://app.example.com", "https://app.example.com/path"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ALLOWED_ORIGINS", value)
			if _, err := LoadConfig(); err == nil {
				t.Fatalf("invalid ALLOWED_ORIGINS %q accepted", value)
			}
		})
	}
}

func TestLoadConfigRejectsTestSMSInProduction(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SMS_DRIVER", "test")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("test SMS driver accepted in production")
	}
}
