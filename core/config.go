package core

import (
	"time"
)

// Server configuration
type Server struct {
	Port   int    `yaml:"port"`
	APIKey string `yaml:"api_key"`
}

// TronGrid configuration
type TronGrid struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

// EtherScan configuration
type EtherScan struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

// QuickNode configuration
type QuickNode struct {
	TronEndpoint string `yaml:"tron_endpoint"`
	EthEndpoint  string `yaml:"eth_endpoint"`
	APIKey       string `yaml:"api_key"`
	APIKeyHeader string `yaml:"api_key_header"`
}

// Intervals configuration
type Intervals struct {
	Sync  time.Duration `yaml:"sync_interval"`
	Fetch time.Duration `yaml:"fetch_interval"`
}

// DBConfig instance define
type Config struct {
	Server     Server    `yaml:"server"`
	TronGrid   TronGrid  `yaml:"trongrid"`
	EtherScan  EtherScan `yaml:"etherscan"`
	QuickNode  QuickNode `yaml:"quicknode"`
	Intervals  Intervals `yaml:"intervals"`
	FetchOnly  bool      `yaml:"fetch_only"`
	ConfigPath string    `yaml:"config_path"`
	Otel       Otel      `yaml:"otel"`
}

type AppConfig struct {
	DBDataSource string   `yaml:"db-data-source"`
	DBConfig     DBConfig `yaml:"db-config"`
	DomainURL    string   `yaml:"domain-url"`
	Otel         Otel     `yaml:"otel"`
	Redis        Redis    `yaml:"redis"`
}
type DBConfig struct {
	MaxConn     int `yaml:"maxConn"`
	IdleConn    int `yaml:"idleConn"`
	MaxLeftTime int `yaml:"maxLeftTime"`
}

type Redis struct {
	Url      string `yaml:"url"`
	Name     string `yaml:"name"`
	Database int    `yaml:"database"`
	Password string `yaml:"password"`
}

type Otel struct {
	Service   string    `yaml:"service"`
	Endpoints Endpoints `yaml:"endpoints"`
}

type Endpoints struct {
	HTTP string `yaml:"http"`
	GRPC string `yaml:"grpc"`
}
