package common

import (
	"github.com/goccy/go-yaml"
	"os"
)

type Config struct {
	GithubRepo string `yaml:"githubRepo"`
}

func LoadConfig() (*Config, error) {
	config := &Config{
		GithubRepo: "github.com/myuser/go-sfml",
	}

	filepath := "./config.yml"
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
