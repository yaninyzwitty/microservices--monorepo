package pkg

import (
	"io"
	"log/slog"

	"gopkg.in/yaml.v3"
)

type Config struct {
	QqlgenServer   QqlgenServer   `yaml:"qqlgen_server"`
	ProductsServer ProductsServer `yaml:"products_server"`
	Astra          Database       `yaml:"database"`
}

type QqlgenServer struct {
	Port int `yaml:"port"`
}

type ProductsServer struct {
	Port int `yaml:"port"`
}

type Database struct {
	Uri   string `yaml:"uri"`
	Topic string `yaml:"topic"`
	// token
}

func (c *Config) LoadConfig(file io.Reader) error {
	data, err := io.ReadAll(file)
	if err != nil {
		slog.Error("Failed to read file", "error", err)
		return err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		slog.Error("Failed to unmarshal data", "error", err)
		return err
	}
	return nil

}
