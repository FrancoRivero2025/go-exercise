package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/FrancoRivero2025/go-exercise/internal/adapters/log"
	"github.com/FrancoRivero2025/go-exercise/internal/domain"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig  `yaml:"server"`
	Pairs    []domain.Pair `yaml:"pairs"`
	Cache    CacheConfig   `yaml:"cache"`
	Kraken   KrakenConfig  `yaml:"kraken"`
	LogLevel int           `yaml:"logLevel"`
	LogPath  string        `yaml:"logOutput"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type CacheConfig struct {
	TTL int `yaml:"ttl"`
}

type KrakenConfig struct {
	URL string `yaml:"url"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
)

func Initialize(path string) *Config {
	once.Do(func() {
		mu.Lock()
		defer mu.Unlock()

		config := Default()

		logger := log.GetInstance()
		
		logger.SetLevel(config.LogLevel)

		logger.Info("Initializing configuration...")
		logger.Info("Config file path: %s", path)

		defer func() {
			if r := recover(); r != nil {
				logger.Error("Error loading settings: %v. Using default setting", r)
				instance = &config

				if config.LogPath != "" {
					if err := logger.SetOutputToFile(config.LogPath); err != nil {
						logger.Warn("Failed to set log file: %v. Using stdout", err)
					}
				}
				logger.SetLevel(config.LogLevel)
			}
		}()

		if _, err := os.Stat(path); os.IsNotExist(err) {
			logger.Warn("Config file not found: %s. Using default setting", path)
			instance = &config
			
			if config.LogPath != "" {
				if err := logger.SetOutputToFile(config.LogPath); err != nil {
					logger.Warn("Failed to set log file: %v. Using stdout", err)
				}
			}
			logger.SetLevel(config.LogLevel)
			return
		}

		content, err := os.ReadFile(path)
		if err != nil {
			panic("Error reading config file: " + err.Error())
		}

		if err := yaml.Unmarshal(content, &config); err != nil {
			panic("Error deserializing YAML configuration: " + err.Error())
		}

		config.Validate()
		instance = &config
		
		if config.LogPath != "" {
			if err := logger.SetOutputToFile(config.LogPath); err != nil {
				logger.Warn("Failed to set log file: %v. Using stdout", err)
			}
		}
		logger.SetLevel(config.LogLevel)
		
		logger.Info("Configuration loaded successfully")
		logger.Debug("Configuration: %+v", config)
	})

	return instance
}

func GetInstance() *Config {
	if instance == nil {
		log.GetInstance().Error("Configuration not initialized. Call Initialize() first")
		panic("Configuration not initialized. Call Initialize() first")
	}
	return instance
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Pairs: []domain.Pair{"BTC/USD", "BTC/EUR", "BTC/CHF"},
		Cache: CacheConfig{
			TTL: 60,
		},
		Kraken: KrakenConfig{
			URL: "https://api.kraken.com",
		},
		LogLevel: 0,
		LogPath:  "/tmp/app.log",
	}
}

func Load(path string) Config {
	config := Default()

	defer func() {
		if r := recover(); r != nil {
			log.GetInstance().Info("Using default settings")
		}
	}()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config
	}

	content, err := os.ReadFile(path)
	if err != nil {
		panic("Error reading config file: " + err.Error())
	}

	if err := yaml.Unmarshal(content, &config); err != nil {
		panic("Error deserializing YAML configuration: " + err.Error())
	}

	return config
}

func (c Config) Validate() {
	logger := log.GetInstance()
	
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errMsg := fmt.Sprintf("Server invalid Port: %d", c.Server.Port)
		logger.Error(errMsg)
		panic(errMsg)
	}

	if c.Cache.TTL <= 0 {
		errMsg := "Cache TTL must be positive"
		logger.Error(errMsg)
		panic(errMsg)
	}

	if len(c.Pairs) == 0 {
		errMsg := "Must specify at least one trading pair"
		logger.Error(errMsg)
		panic(errMsg)
	}

	if c.LogLevel < 0 || c.LogLevel > 4 {
		logger.Warn("Invalid log level: %d. Using default level", c.LogLevel)
		c.LogLevel = 1
	}
}
