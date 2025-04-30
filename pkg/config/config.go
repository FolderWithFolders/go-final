package config

import (
	"errors"
	"os"
)

type Config struct {
	Port     string
	Password string
}

func Load() (*Config, error) {
	password := os.Getenv("TODO_PASSWORD")
	if password == "" {
		return nil, errors.New("переменная окружения TODO_PASSWORD обязательна")
	}

	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	return &Config{
		Port:     port,
		Password: password,
	}, nil
}
