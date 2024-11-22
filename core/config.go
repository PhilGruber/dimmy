package core

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

func LoadConfig() (*ServerConfig, error) {

	var filename string
	if _, err := os.Stat("/etc/dimmy/dimmyd.conf.yaml"); err == nil {
		filename = "/etc/dimmy/dimmyd.conf.yaml"
	} else if _, err := os.Stat("dimmyd.conf.yaml"); err == nil {
		filename = "dimmyd.conf.yaml"
	} else {
		return nil, errors.New("could not find config file /etc/dimmy/dimmyd.conf.yaml")
	}

	log.Println("Loading config file " + filename)

	var config ServerConfig
	configYaml, _ := os.ReadFile(filename)
	err := yaml.Unmarshal(configYaml, &config)
	if err != nil {
		log.Fatal(err)
	}

	if config.WebRoot == "" {
		config.WebRoot = "/usr/share/dimmy"
	}

	if config.MqttServer == "" {
		config.MqttServer = "127.0.0.1"
	}

	if config.Port == 0 {
		config.Port = 80
	}

	return &config, nil
}
