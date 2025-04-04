package config

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"log"
	"strings"
)

type AppConfig struct {
	Port int
	Auth struct {
		Secret string
		Issuer string
	}
	Rabbit struct {
		URL         string
		Connections uint32
		Channels    uint32
	}
	Server struct {
		ID   uuid.UUID
		Host string
	}
}

var Config AppConfig

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/app/config")
	viper.SetEnvPrefix("FARKLE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("Port", 8080)
	viper.SetDefault("Auth.Secret", "my-secret")
	viper.SetDefault("Auth.Issuer", "dice-farkle-server")
	viper.SetDefault("Rabbit.URL", "amqp://guest:guest@localhost:5672")
	viper.SetDefault("Rabbit.Connections", 10)
	viper.SetDefault("Rabbit.Channels", 10)
	viper.SetDefault("Server.ID", uuid.New())
	viper.SetDefault("Server.Host", "localhost:8080")

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
