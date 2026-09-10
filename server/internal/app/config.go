package app

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	AllowedOrigins []string
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
	return Config{HTTPAddr: address, DatabaseURL: databaseURL, AllowedOrigins: origins}, nil
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
