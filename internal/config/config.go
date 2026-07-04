package config

import "time"

type Document struct {
	Database DatabaseConfig
	Queries  []Query
}

type DatabaseConfig struct {
	Type                  string        `yaml:"type"`
	Host                  string        `yaml:"host"`
	Port                  int           `yaml:"port"`
	Database              string        `yaml:"database"`
	Schema                string        `yaml:"schema"`
	ServiceName           string        `yaml:"service_name"`
	SID                   string        `yaml:"sid"`
	Username              string        `yaml:"username"`
	Password              string        `yaml:"password"`
	SSLMode               string        `yaml:"sslmode"`
	ConnectTimeoutSeconds int           `yaml:"connect_timeout_seconds"`
	ConnectTimeout        time.Duration `yaml:"-"`
}

type Query struct {
	Name string
	SQL  string
}
