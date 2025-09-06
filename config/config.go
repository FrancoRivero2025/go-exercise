package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Pairs []string `yaml:"pairs"`
	Cache struct {
		TTL int `yaml:"ttl"`
	} `yaml:"cache"`
	Kraken struct {
		URL string `yaml:"url"`
	} `yaml:"kraken"`
}

func Default() Config {
	var c Config
	c.Server.Port = 8080
	c.Pairs = []string{"BTC/USD", "BTC/EUR", "BTC/CHF"}
	c.Cache.TTL = 60
	c.Kraken.URL = "https://api.kraken.com"
	return c
}

func (c Config) ServerPort() int { return c.Server.Port }
func (c Config) ServerPortString() string { return fmt.Sprintf("%d", c.Server.Port) }

// Helper to transform to domain pairs (used by refresher)
func (c Config) PairsAsDomain() []string { return c.Pairs }

func Load(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}
