package config

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os"
)

var Cfg = NewConfig("./config/config.json")

type Config struct {
	Conn struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
	Login struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	}
}

func NewConfig(filePath string) *Config {
	var config Config
	data, err := os.ReadFile(filePath)
	if err != nil {
		logrus.Fatal(err)
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		logrus.Fatal(err)

	}
	return &config
}
