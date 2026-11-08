package config

import (
	"errors"
	"io/fs"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Load fills T from the environment, after loading a .env file from the
// working directory when one exists.
func Load[T any]() (T, error) {
	var cfg T
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cfg, err
	}
	return cfg, env.Parse(&cfg)
}
