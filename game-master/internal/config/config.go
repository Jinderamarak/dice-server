package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
)

type AppConfig struct {
	Port      int
	RabbitURL string
}

var Config AppConfig

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/app/config")
	viper.SetEnvPrefix("DICE")
	viper.AutomaticEnv()

	viper.SetDefault("Port", 8080)
	viper.SetDefault("RabbitURL", "amqp://guest:guest@localhost:5672")

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
