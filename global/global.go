package global

import (
	"github.com/philchia/agollo/v4"
	"github.com/spf13/viper"
	"log"
	"quicknode/core"
	"strings"
	"time"
)

var (
	GConfig *Config // Configuration information
)

type Config struct {
	Viper      *viper.Viper
	YamlConfig *YamlConfig
}

type YamlConfig struct {
	DBConfig DBConfig `yaml:"db-config"`
}

type DBConfig struct {
	MaxConn     int `yaml:"maxConn"`
	IdleConn    int `yaml:"idleConn"`
	MaxLeftTime int `yaml:"maxLeftTime"`
}

func New(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	err := v.ReadInConfig()
	if err != nil {
		log.Printf("Failed to read config file: %s\n", err)
		return nil, err
	}
	GConfig = &Config{Viper: v}
	return GConfig, err
}
func NewFromViper(v *viper.Viper) (*Config, error) {
	GConfig = &Config{Viper: v}
	return GConfig, nil
}

func NewFromApollo(namespace string) error {
	// Get configuration from Apollo
	apolloConfig := agollo.GetContent(agollo.WithNamespace(namespace))

	// Create a new Viper instance
	v := viper.New()
	v.SetConfigType("yaml")

	// Load the Apollo configuration into Viper
	err := v.ReadConfig(strings.NewReader(apolloConfig))
	if err != nil {
		log.Printf("Failed to read Apollo config: %s\n", err)
		return err
	}

	// Set the global configuration
	GConfig = &Config{Viper: v}

	return nil
}

func InitGlobal() func() {
	dbConfig := core.DBConfig{
		MaxConn:      GConfig.Viper.GetInt("db-config.maxConn"),
		IdleConn:     GConfig.Viper.GetInt("db-config.idleConn"),
		MaxLeftTime:  GConfig.Viper.GetInt("db-config.maxLeftTime"),
		DBDataSource: GConfig.Viper.GetString("db-config.db-data-source"),
	}

	log.Printf("InitGlobal: MaxConn=%d, IdleConn=%d, MaxLeftTime=%d, DBDataSource=%s",
		dbConfig.MaxConn, dbConfig.IdleConn, dbConfig.MaxLeftTime, dbConfig.DBDataSource)

	err := AddDB(DBNameToken13, dbConfig.DBDataSource, dbConfig.MaxConn, dbConfig.IdleConn, time.Duration(dbConfig.MaxLeftTime)*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return nil
	}
	//for local dev
	//AddDB("user", "root:root@tcp(127.0.0.1:3306)/user?charset=utf8mb4&parseTime=True&loc=Local")
	return ReleaseMysqlDBPool
}
