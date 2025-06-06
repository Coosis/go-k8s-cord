package model

import (
	"fmt"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	AGENT_CONFIG = "agent_config.yaml"

	DEFAULT_HTTPS_CENTRAL      = "https://localhost:10201"
	DEFAULT_GRPC_CENTRAL       = "localhost:10202"
	DEFAULT_HTTPS_PORT         = ":10203"
	DEFAULT_GRPC_PORT          = ":10204"
	DEFAULT_HEARTBEAT_INTERVAL = 3
	DEFAULT_DEPLOYMENT_DIR     = "agent_deployments"

	// HEARTBEAT_PATH = "/heartbeat"
	// REGISTER_PATH  = "/register"
	ADDR_PLACEHOLDER = "CHANGE_ME_TO_AGENT_ENDPOINT"
	EDIT_CONFIG_MESSAAGE = "Please edit the config file " +
	"before re-starting the agent. Make sure to set the agent's address so " +
	"that the central server can communicate with it. Make sure it matches the " +
	"field in the certificate."
	AGENT_CONFIG_NOT_SET = "Agent address is not set in the config file. " +
	"Please edit the config file and set the address before continuing."
)

var config atomic.Value

func GetAgentConfig() *AgentConfig {
	cfg := config.Load()
	if cfg == nil {
		log.Error("Agent config is not initialized")
		return nil
	}
	return cfg.(*AgentConfig)
}

func WriteBackAndReload() error {
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config after registration: %w", err)
	}
	log.Info("Agent config file written successfully, reloading...")
	return nil;
}

func init() {
	if err := AgentServerConfigSetup(); err != nil {
		log.Fatal("Failed to setup agent server config: ", err)
		panic(err)
	}
}

type AgentConfig struct {
	AgentName         string `yaml:"agent_name"`
	Addr              string `yaml:"addr"`
	Version           string `yaml:"version"`
	GrpcCentral       string `yaml:"grpc_central"`
	HTTPSCentral      string `yaml:"https_central"`
	UUID              string `yaml:"uuid"`
	// An entry indicating that the agent is already registered
	// Not present if the agent is not registered
	Registered        bool   `yaml:"registered"`
	HTTPSPort         string `yaml:"https_port"`
	GRPCPort          string `yaml:"grpc_port"`
	HeartbeatInterval int    `yaml:"heartbeat_interval"`
	DeploymentDir     string `yaml:"deployment_dir"`
}

func NewAgentConfig() *AgentConfig {
	return &AgentConfig{
		AgentName:         viper.GetString("agent_name"),
		Addr:              viper.GetString("addr"),
		Version:           viper.GetString("version"),
		GrpcCentral:       viper.GetString("grpc_central"),
		HTTPSCentral:      viper.GetString("https_central"),
		UUID:              viper.GetString("uuid"),
		Registered:        viper.GetBool("registered"),
		HTTPSPort:         viper.GetString("https_port"),
		GRPCPort:          viper.GetString("grpc_port"),
		HeartbeatInterval: viper.GetInt("heartbeat_interval"),
		DeploymentDir:     viper.GetString("deployment_dir"),
	}
}

func AgentServerConfigSetup() error {
	// loading agent config
	viper.SetConfigName(AGENT_CONFIG)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	viper.SetDefault("agent_name", "default_agent")
	viper.SetDefault("addr", ADDR_PLACEHOLDER)
	viper.SetDefault("version", "1.0.0")
	viper.SetDefault("grpc_central", DEFAULT_GRPC_CENTRAL)
	viper.SetDefault("grpc_port", DEFAULT_GRPC_PORT)
	viper.SetDefault("https_central", DEFAULT_HTTPS_CENTRAL)
	viper.SetDefault("https_port", DEFAULT_HTTPS_PORT)
	viper.SetDefault("uuid", "")
	viper.SetDefault("registered", false)
	viper.SetDefault("heartbeat_interval", DEFAULT_HEARTBEAT_INTERVAL)
	viper.SetDefault("deployment_dir", DEFAULT_DEPLOYMENT_DIR)

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed: ", e.Name)
		cfg := NewAgentConfig()
		config.Store(cfg)
		log.Info("Reloaded config for agent: ", cfg.AgentName)
	})
	viper.WatchConfig()

	err := viper.ReadInConfig()
	if err != nil {
		if _, notfound := err.(viper.ConfigFileNotFoundError); notfound {
			log.Info("Agent config file not found, creating a new one...")
			err = viper.WriteConfigAs(AGENT_CONFIG)
			if err != nil {
				log.Error("Failed to create agent config file: ", err)
				return err
			}
			log.Info("Agent config file created successfully")
		} else {
			log.Error("Failed to read agent config: ", err)
			return err
		}
	}

	cfg := NewAgentConfig()
	if cfg.Addr == ADDR_PLACEHOLDER {
		log.Fatal(AGENT_CONFIG_NOT_SET)
		panic(AGENT_CONFIG_NOT_SET)
	}
	config.Store(cfg)

	return nil
}

func (c *AgentConfig) GRPCEndpoint() string {
	return c.Addr + c.GRPCPort
}
