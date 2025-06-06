// Central config file read/write & file watch, along with other configs
package model

import (
	"sync/atomic"
	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"

	log "github.com/sirupsen/logrus"
)

const (
	CENTRAL_CONFIG         = "central_config.yaml"
	DEFAULT_ALIVE_INTERVAL = 5 // seconds
	DEFAULT_ETCD_ADDR      = "localhost:2379"
	DEFAULT_HTTPS_PORT     = ":10201"
	DEFAULT_GRPC_PORT      = ":10202"
	HTTPS_LOCALHOST        = "https://localhost" + DEFAULT_HTTPS_PORT
	GRPC_LOCALHOST         = "https://localhost" + DEFAULT_GRPC_PORT
	LOCAL_STATUS_URL       = HTTPS_LOCALHOST + "/status"

	// git related constants
	DEFAULT_DEPLOYMENTS_DIR = "deployments"
	DEFAULT_GIT_REMOTE      = "origin"
	DEFAULT_GIT_BRANCH      = "main"
	DEFAULT_GIT_REMOTE_URL  = "<REPLACE_WITH_GIT_REMOTE_URL>"
)

// used globally
var config atomic.Value

func GetCentralConfig() *CentralConfig {
	cfg := config.Load()
	if cfg == nil {
		log.Error("Central config is not initialized")
		return nil
	}
	return cfg.(*CentralConfig)
}

func init() {
	if err := CentralServerConfigSetup(); err != nil {
		log.Fatal("Failed to setup central server config: ", err)
		panic(err)
	}
}

type CentralConfig struct {
	AliveInterval int    `yaml:"alive_interval"`
	Version       string `yaml:"version"`
	HTTPSPort     string `yaml:"https_port"`
	GRPCPort      string `yaml:"grpc_port"`
	EtcdAddr      string `yaml:"etcd_addr"`

	DeploymentsDir string `yaml:"deployments_dir"`
	GitRemoteName  string `yaml:"git_remote_name"`
	GitBranch      string `yaml:"git_branch"`
}

func NewCentralConfig() *CentralConfig {
	return &CentralConfig{
		AliveInterval: viper.GetInt("alive_interval"),
		Version:       viper.GetString("version"),
		HTTPSPort:     viper.GetString("https_port"),
		GRPCPort:      viper.GetString("grpc_port"),
		EtcdAddr:      viper.GetString("etcd_addr"),

		DeploymentsDir: viper.GetString("deployments_dir"),
		GitRemoteName:  viper.GetString("git_remote_name"),
		GitBranch:      viper.GetString("git_branch"),
	}
}

func CentralServerConfigSetup() error {
	// ------ IMPORTANT: ALSO UPDATE CentralConfig ------
	// loading agent config
	viper.SetConfigName(CENTRAL_CONFIG)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	viper.SetDefault("version", "1.0.0")
	viper.SetDefault("alive_interval", DEFAULT_ALIVE_INTERVAL)
	viper.SetDefault("https_port", DEFAULT_HTTPS_PORT)
	viper.SetDefault("grpc_port", DEFAULT_GRPC_PORT)
	viper.SetDefault("etcd_addr", DEFAULT_ETCD_ADDR)

	viper.SetDefault("deployments_dir", DEFAULT_DEPLOYMENTS_DIR)
	viper.SetDefault("git_remote_name", DEFAULT_GIT_REMOTE)
	viper.SetDefault("git_branch", DEFAULT_GIT_BRANCH)

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed: ", e.Name)
		cfg := NewCentralConfig()
		config.Store(cfg)
		log.Info("Reloaded config for central")
	})
	viper.WatchConfig()

	err := viper.ReadInConfig()
	if err != nil {
		if _, notfound := err.(viper.ConfigFileNotFoundError); notfound {
			log.Info("Central config file not found, creating a new one...")
			err = viper.WriteConfigAs(CENTRAL_CONFIG)
			if err != nil {
				log.Error("Failed to create central config file: ", err)
				return err
			}
			log.Info("Central config file created successfully")
		} else {
			log.Error("Failed to read central config: ", err)
			return err
		}
	}
	err = viper.WriteConfig()
	if err != nil {
		log.Error("Failed to write central config: ", err)
		return err
	}

	cfg := NewCentralConfig()
	config.Store(cfg)

	return nil
}
