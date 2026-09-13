package app

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	AllowedOrigins []string
	Environment    string
	SMSDriver      string
	StorageDriver  string
	StoragePath    string
	ImageURLSecret string
	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	OpenAIAPIKey   string
	OpenAIModel    string
	OpenAIBaseURL  string
	OpenAIAPIStyle string
	OpenAITimeout  time.Duration
}

func LoadConfig() (Config, error) {
	address, exists := os.LookupEnv("HTTP_ADDR")
	if !exists {
		address = "127.0.0.1:8080"
	}
	_, port, err := net.SplitHostPort(address)
	number, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || number < 1 || number > 65535 {
		return Config{}, errors.New("HTTP_ADDR must be host:port with a port between 1 and 65535")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	parsedDatabaseURL, err := url.Parse(databaseURL)
	if err != nil || (parsedDatabaseURL.Scheme != "postgres" && parsedDatabaseURL.Scheme != "postgresql") || parsedDatabaseURL.Host == "" || parsedDatabaseURL.Path == "" || parsedDatabaseURL.Path == "/" {
		return Config{}, errors.New("DATABASE_URL must be a PostgreSQL connection URL with a database name")
	}
	originsValue, exists := os.LookupEnv("ALLOWED_ORIGINS")
	if !exists {
		originsValue = "http://127.0.0.1:5173"
	}
	origins, err := parseOrigins(originsValue)
	if err != nil {
		return Config{}, err
	}
	environment := envOrDefault("APP_ENV", "development")
	smsDriver := envOrDefault("SMS_DRIVER", "test")
	storageDriver := envOrDefault("STORAGE_DRIVER", "filesystem")
	if storageDriver != "filesystem" && storageDriver != "s3" {
		return Config{}, errors.New("STORAGE_DRIVER must be filesystem or s3")
	}
	if environment == "production" && storageDriver != "s3" {
		return Config{}, errors.New("filesystem storage is forbidden in production")
	}
	storagePath := envOrDefault("STORAGE_PATH", "./var/images")
	imageURLSecret := envOrDefault("IMAGE_URL_SECRET", "formtally-development-image-secret")
	s3Values := []string{os.Getenv("S3_ENDPOINT"), os.Getenv("S3_REGION"), os.Getenv("S3_BUCKET"), os.Getenv("S3_ACCESS_KEY"), os.Getenv("S3_SECRET_KEY")}
	if storageDriver == "s3" {
		for _, value := range s3Values {
			if value == "" {
				return Config{}, errors.New("S3 configuration is incomplete")
			}
		}
	}
	if smsDriver != "test" {
		return Config{}, errors.New("SMS_DRIVER is not supported")
	}
	if environment == "production" && smsDriver == "test" {
		return Config{}, errors.New("test SMS driver is forbidden in production")
	}
	openAIBaseURL := envOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1")
	parsedOpenAIBaseURL, err := url.Parse(openAIBaseURL)
	if err != nil || (parsedOpenAIBaseURL.Scheme != "http" && parsedOpenAIBaseURL.Scheme != "https") || parsedOpenAIBaseURL.Host == "" || parsedOpenAIBaseURL.User != nil || parsedOpenAIBaseURL.RawQuery != "" || parsedOpenAIBaseURL.Fragment != "" {
		return Config{}, errors.New("OPENAI_BASE_URL must be an HTTP base URL without credentials, query, or fragment")
	}
	openAIAPIStyle := envOrDefault("OPENAI_API_STYLE", "responses")
	if openAIAPIStyle != "responses" && openAIAPIStyle != "chat_completions" {
		return Config{}, errors.New("OPENAI_API_STYLE must be responses or chat_completions")
	}
	openAITimeout, err := time.ParseDuration(envOrDefault("OPENAI_TIMEOUT", "20s"))
	if err != nil || openAITimeout <= 0 {
		return Config{}, errors.New("OPENAI_TIMEOUT must be a positive duration")
	}
	return Config{
		HTTPAddr: address, DatabaseURL: databaseURL, AllowedOrigins: origins, Environment: environment, SMSDriver: smsDriver,
		StorageDriver: storageDriver, StoragePath: storagePath, ImageURLSecret: imageURLSecret,
		S3Endpoint: s3Values[0], S3Region: s3Values[1], S3Bucket: s3Values[2], S3AccessKey: s3Values[3], S3SecretKey: s3Values[4],
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"), OpenAIModel: os.Getenv("OPENAI_MODEL"), OpenAIBaseURL: openAIBaseURL, OpenAIAPIStyle: openAIAPIStyle, OpenAITimeout: openAITimeout,
	}, nil
}

func envOrDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func parseOrigins(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, errors.New("ALLOWED_ORIGINS must contain comma-separated HTTP origins without paths")
		}
		origins = append(origins, origin)
	}
	return origins, nil
}
