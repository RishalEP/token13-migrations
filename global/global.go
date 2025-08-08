package global

import (
	"github.com/spf13/viper"
	"log"
	"quicknode/core"
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

func InitGlobal() func() {
	config := &core.AppConfig{
		DBDataSource: GConfig.Viper.GetString("db-data-source"),
		DBConfig: core.DBConfig{
			MaxConn:     GConfig.Viper.GetInt("db-config.maxConn"),
			IdleConn:    GConfig.Viper.GetInt("db-config.idleConn"),
			MaxLeftTime: GConfig.Viper.GetInt("db-config.maxLeftTime"),
		},
	}
	err := AddDB(DBNameToken13, config.DBDataSource, config.DBConfig.MaxConn, config.DBConfig.IdleConn, time.Duration(config.DBConfig.MaxLeftTime)*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return nil
	}
	//for local dev
	//AddDB("user", "root:root@tcp(127.0.0.1:3306)/user?charset=utf8mb4&parseTime=True&loc=Local")
	return ReleaseMysqlDBPool
}
