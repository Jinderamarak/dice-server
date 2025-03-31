package config

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"net/url"
)

type AppConfig struct {
	Port      int
	RabbitURL url.URL
	ServerID  uuid.UUID
	ServerURL url.URL
}

var Config AppConfig

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/app/config")
	viper.SetEnvPrefix("FARKLE")
	viper.AutomaticEnv()

	port := 8080
	rabbitURL, _ := url.Parse("amqp://guest:guest@localhost:5672")
	serverID := uuid.New()
	serverURL, _ := url.Parse("ws://localhost:8080/game/farkle")

	viper.SetDefault("Port", port)
	viper.SetDefault("RabbitURL", rabbitURL)
	viper.SetDefault("ServerID", serverID)
	viper.SetDefault("ServerURL", serverURL)

	if err := viper.SafeWriteConfig(); err != nil {
		log.Println("Failed to write default config file:", err)
	}

	if err := viper.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			log.Println("Failed to load config file:", err)
		} else {
			return errors.Wrap(err, "failed to read config file")
		}
	}

	if err := viper.Unmarshal(&Config); err != nil {
		return errors.Wrap(err, "failed to unmarshal config")
	}

	return nil
}
