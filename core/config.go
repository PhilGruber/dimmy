package core

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

func LoadConfig() (*ServerConfig, error) {

	var filename string
	var rulesFile string
	if _, err := os.Stat("/etc/dimmy/dimmyd.conf.yaml"); err == nil {
		filename = "/etc/dimmy/dimmyd.conf.yaml"
		rulesFile = "/etc/dimmy/rules.conf.yaml"
	} else if _, err := os.Stat("dimmyd.conf.yaml"); err == nil {
		filename = "dimmyd.conf.yaml"
		rulesFile = "rules.conf.yaml"
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

	if _, err := os.Stat(rulesFile); err == nil {
		log.Println("Loading rules file " + rulesFile)
		rulesYaml, _ := os.ReadFile(rulesFile)
		err := yaml.Unmarshal(rulesYaml, &config.Rules)
		if err != nil {
			log.Println("Could not load rules.conf.yaml: " + err.Error())
		}
	} else {
		log.Println("Could not find rules file " + rulesFile)
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
