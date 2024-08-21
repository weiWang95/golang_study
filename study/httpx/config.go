package httpx

import (
	"time"
)

type Config struct {
	Host      string
	UrlPrefix string
	Timeout   time.Duration
}

func defaultCfg() *Config {
	return &Config{
		Timeout: 5 * time.Second,
	}
}

type configFn func(cfg *Config)

func WithHost(host string) configFn {
	return func(cfg *Config) {
		cfg.Host = host
	}
}

func WithUrlPrefix(prefix string) configFn {
	return func(cfg *Config) {
		cfg.UrlPrefix = prefix
	}
}

func WithTimeout(timeout time.Duration) configFn {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}
