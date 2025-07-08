package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	IPAddr                string `json:"IPAddr"`
	RPCEndpoint           string `json:"rpcEndpoint"`
	DispatcherAddr        string `json:"dispatcherAddr"`
	DispatcherABIFilePath string `json:"dispatcherABIFilePath"`
	PrivateKey            string `json:"privateKey"`
}

func Load() (Config, error) {
	var config Config

	configBytes, err := os.ReadFile("config.json")
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(configBytes, &config)

	return config, nil
}
