package app

import (
	"errors"
	"net"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr string
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
	return Config{HTTPAddr: address}, nil
}
