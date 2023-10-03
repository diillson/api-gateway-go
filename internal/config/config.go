package config

import (
	"errors"
	"os"
)

type Config struct {
	Port string
}

func LoadConfig() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		return nil, errors.New("PORT is not set")
	}

	return &Config{
		Port: port,
	}, nil
}
