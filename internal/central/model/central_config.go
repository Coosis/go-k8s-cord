package model

import "github.com/spf13/viper"

const (
	CENTRAL_CONFIG = "central_config.yaml"
	DEFAULT_ALIVE_INTERVAL = 5 // seconds
	DEFAULT_ETCD_ADDR = "localhost:2379"
	DEFAULT_HTTPS_PORT = ":10201"
	DEFAULT_GRPC_PORT = ":10202"
	HTTPS_LOCALHOST = "https://localhost" + DEFAULT_HTTPS_PORT
	GRPC_LOCALHOST = "https://localhost" + DEFAULT_GRPC_PORT
	LOCAL_STATUS_URL = HTTPS_LOCALHOST + "/status"
)

type CentralConfig struct {
	AliveInterval            int    `yaml:"alive_interval"`
	Version                  string `yaml:"version"`
	HTTPSPort                string `yaml:"https_port"`
	GRPCPort                 string `yaml:"grpc_port"`
}

func NewCentralConfig() *CentralConfig {
	return &CentralConfig{
		AliveInterval:            viper.GetInt("alive_interval"),
		Version:                  viper.GetString("version"),
		HTTPSPort:                viper.GetString("https_port"),
		GRPCPort:                 viper.GetString("grpc_port"),
	}
}
