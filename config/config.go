package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Grpc_server struct {
		Host    string `yaml:"host"`
		Port    string `yaml:"port"`
		Network string `yaml:"network"`
	} `yaml:"grpc_server"`

	Log_level string `yaml:"log_level"`
}

func GetConfig() (*Config, error) {
	data, err := ioutil.ReadFile("../config.yaml")
	if err != nil {
		return nil, err
	}

	var config Config

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
