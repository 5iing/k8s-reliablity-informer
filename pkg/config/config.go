package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Checker struct {
		CheckPods        bool `yaml:"check_pods"`
		CheckNodes       bool `yaml:"check_nodes"`
		CheckDeployments bool `yaml:"check_deployments"`
	} `yaml:"checker"`

	Notifiers struct {
		Discord struct {
			Enabled    bool   `yaml:"enabled"`
			WebhookURL string `yaml:"webhook_url"`
		} `yaml:"discord"`

		Console struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"console"`
	} `yaml:"notifiers"`
}

func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config AppConfig
	err = yaml.Unmarshal(data, &config)
	return &config, err
}
