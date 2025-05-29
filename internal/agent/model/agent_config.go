package model

import "github.com/spf13/viper"

const (
	AGENT_CONFIG = "agent_config.yaml"

	DEFAULT_HTTPS_CENTRAL      = "https://localhost:10201"
	DEFAULT_GRPC_CENTRAL       = "localhost:10202"
	DEFAULT_HTTPS_PORT         = ":10203"
	DEFAULT_GRPC_PORT          = ":10204"
	DEFAULT_HEARTBEAT_INTERVAL = 3

	// HEARTBEAT_PATH = "/heartbeat"
	// REGISTER_PATH  = "/register"
)

// An entry indicating that the agent is already registered
// Not present if the agent is not registered
type AgentConfig struct {
	AgentName         string `yaml:"agent_name"`
	Addr              string `yaml:"addr"`
	Version           string `yaml:"version"`
	GrpcCentral       string `yaml:"grpc_central"`
	HTTPSCentral      string `yaml:"https_central"`
	UUID              string `yaml:"uuid"`
	Registered        bool   `yaml:"registered"`
	HTTPSPort         string `yaml:"https_port"`
	GRPCPort          string `yaml:"grpc_port"`
	HeartbeatInterval int    `yaml:"heartbeat_interval"`
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
	}
}

func (c *AgentConfig) GRPCEndpoint() string {
	return c.Addr + c.GRPCPort
}

// func (c *AgentConfig) HeartbeatURL() string {
// 	return c.Central + HEARTBEAT_PATH
// }
//
// func (c *AgentConfig) RegisterURL() string {
// 	return c.Central + REGISTER_PATH
// }
