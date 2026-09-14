package app

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
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

func TestLoadConfigRequiresPrivateS3InProductionAndParsesAISettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SMS_DRIVER", "test")
	t.Setenv("STORAGE_DRIVER", "filesystem")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("filesystem storage accepted in production")
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("STORAGE_DRIVER", "s3")
	for name, value := range map[string]string{
		"S3_ENDPOINT": "https://objects.example.com", "S3_REGION": "cn-east-1", "S3_BUCKET": "formtally-private",
		"S3_ACCESS_KEY": "access", "S3_SECRET_KEY": "secret", "OPENAI_API_KEY": "key", "OPENAI_MODEL": "image-capable-model",
		"OPENAI_TIMEOUT": "7s",
	} {
		t.Setenv(name, value)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.S3Bucket != "formtally-private" || cfg.OpenAIModel != "image-capable-model" || cfg.OpenAITimeout != 7*time.Second {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestLoadConfigDefaultsAndOverridesOpenAIBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	original, existed := os.LookupEnv("OPENAI_BASE_URL")
	if err := os.Unsetenv("OPENAI_BASE_URL"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("OPENAI_BASE_URL", original)
		} else {
			_ = os.Unsetenv("OPENAI_BASE_URL")
		}
	})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIBaseURL != "https://api.openai.com/v1" {
		t.Fatalf("default OpenAIBaseURL = %q", cfg.OpenAIBaseURL)
	}

	if err := os.Setenv("OPENAI_BASE_URL", "https://gateway.example.com/openai/v1"); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIBaseURL != "https://gateway.example.com/openai/v1" {
		t.Fatalf("custom OpenAIBaseURL = %q", cfg.OpenAIBaseURL)
	}
}

func TestLoadConfigDefaultsAndValidatesOpenAIAPIStyle(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	original, existed := os.LookupEnv("OPENAI_API_STYLE")
	if err := os.Unsetenv("OPENAI_API_STYLE"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("OPENAI_API_STYLE", original)
		} else {
			_ = os.Unsetenv("OPENAI_API_STYLE")
		}
	})

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIAPIStyle != "responses" {
		t.Fatalf("default OpenAIAPIStyle = %q", cfg.OpenAIAPIStyle)
	}

	if err := os.Setenv("OPENAI_API_STYLE", "chat_completions"); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIAPIStyle != "chat_completions" {
		t.Fatalf("custom OpenAIAPIStyle = %q", cfg.OpenAIAPIStyle)
	}

	if err := os.Setenv("OPENAI_API_STYLE", "legacy"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid OPENAI_API_STYLE accepted")
	}
}

func TestLoadConfigRejectsInvalidOpenAIBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	for _, value := range []string{"", "api.openai.com/v1", "ftp://api.example.com/v1", "https://api.example.com/v1?token=secret"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("OPENAI_BASE_URL", value)
			if _, err := LoadConfig(); err == nil {
				t.Fatalf("invalid OPENAI_BASE_URL %q accepted", value)
			}
		})
	}
}

func TestLoadConfigWeChatDefaultsAndAllowsDisabledDevelopment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	t.Setenv("APP_ENV", "development")
	t.Setenv("WECHAT_APP_ID", "")
	t.Setenv("WECHAT_APP_SECRET", "")
	t.Setenv("WECHAT_API_BASE_URL", "https://api.weixin.qq.com")
	t.Setenv("WECHAT_API_TIMEOUT", "5s")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WeChatAppID != "" || cfg.WeChatAppSecret != "" || cfg.WeChatAPIBaseURL != "https://api.weixin.qq.com" || cfg.WeChatAPITimeout != 5*time.Second {
		t.Fatalf("WeChat config = %+v", cfg)
	}
}

func TestLoadConfigWeChatRequiresCredentialsTogether(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	t.Setenv("APP_ENV", "development")
	t.Setenv("WECHAT_API_BASE_URL", "https://api.weixin.qq.com")
	t.Setenv("WECHAT_API_TIMEOUT", "5s")

	for name, values := range map[string][2]string{
		"missing_secret": {"wx-app", ""},
		"missing_app_id": {"", "wx-secret"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("WECHAT_APP_ID", values[0])
			t.Setenv("WECHAT_APP_SECRET", values[1])
			if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "WECHAT_APP_ID") {
				t.Fatalf("LoadConfig() error = %v", err)
			}
		})
	}
}

func TestLoadConfigWeChatRejectsUnsafeProductionBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	t.Setenv("APP_ENV", "production")
	t.Setenv("WECHAT_APP_ID", "wx-app")
	t.Setenv("WECHAT_APP_SECRET", "wx-secret")
	t.Setenv("WECHAT_API_TIMEOUT", "5s")

	for _, value := range []string{"http://api.weixin.qq.com", "https://wechat.example.com", "https://api.weixin.qq.com/path"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WECHAT_API_BASE_URL", value)
			if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "WECHAT_API_BASE_URL") {
				t.Fatalf("LoadConfig() error = %v", err)
			}
		})
	}
}

func TestLoadConfigWeChatValidatesPositiveTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://formtally:formtally@127.0.0.1:5432/formtally?sslmode=disable")
	t.Setenv("WECHAT_APP_ID", "")
	t.Setenv("WECHAT_APP_SECRET", "")
	t.Setenv("WECHAT_API_BASE_URL", "https://api.weixin.qq.com")

	for _, value := range []string{"invalid", "0s", "-1s"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WECHAT_API_TIMEOUT", value)
			if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "WECHAT_API_TIMEOUT") {
				t.Fatalf("LoadConfig() error = %v", err)
			}
		})
	}
}
