package pkg

import (
	"io"
	"log/slog"

	"gopkg.in/yaml.v3"
)

type Config struct {
	QqlgenServer       QqlgenServer       `yaml:"gqlgen-server"`
	GrpcServer         GrpcServer         `yaml:"products-server"`
	ProductsServer     ProductsServer     `yaml:"products_server"`
	Roach              Database           `yaml:"database"`
	Queue              Queue              `yaml:"queue"`
	Processer          Processer          `yaml:"processer-server"`
	NotificationServer NotificationServer `yaml:"notification-server"`
	StockServer        StockServer        `yaml:"stock-server"`
	UserServer         UserServer         `yaml:"user-server"`
	OrderServer        OrderServer        `yaml:"order-server"`
}

type OrderServer struct {
	Port int `yaml:"port"`
}
type StockServer struct {
	Port int `yaml:"port"`
}
type UserServer struct {
	Port int `yaml:"port"`
}

type NotificationServer struct {
	Port          int    `yaml:"port"`
	ConsumerTopic string `yaml:"consumer_topic"`
}

type Queue struct {
	Uri   string `yaml:"uri"`
	Topic string `yaml:"topic"`
}

type QqlgenServer struct {
	Port int `yaml:"port"`
}
type Processer struct {
	Port int `yaml:"port"`
}
type GrpcServer struct {
	Port int `yaml:"port"`
}

type ProductsServer struct {
	Port int `yaml:"port"`
}

type Database struct {
	Username   string `yaml:"username"`
	Host       string `yaml:"host"`
	DbName     string `yaml:"database"`
	Port       int    `yaml:"port"`
	SSLMode    string `yaml:"sslmode"`
	MaxRetries int    `yaml:"max_retries"`
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
