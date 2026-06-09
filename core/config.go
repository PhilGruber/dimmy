package core

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	config.Filename = filename

	return &config, nil
}

func AddDeviceToConfig(filename string, device DeviceConfig) error {
	configYaml, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var document yaml.Node
	if err := yaml.Unmarshal(configYaml, &document); err != nil {
		return err
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return errors.New("config file must contain a YAML mapping")
	}

	root := document.Content[0]
	var devices *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "devices" {
			devices = root.Content[i+1]
			break
		}
	}
	if devices == nil {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "devices"},
			&yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"},
		)
		devices = root.Content[len(root.Content)-1]
	}
	if devices.Kind != yaml.SequenceNode {
		return errors.New("config devices must be a YAML sequence")
	}

	var encoded yaml.Node
	if err := encoded.Encode(device); err != nil {
		return err
	}
	devices.Content = append(devices.Content, &encoded)

	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(filename), "."+filepath.Base(filename)+".*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	encoder := yaml.NewEncoder(temp)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		temp.Close()
		return err
	}
	if err := encoder.Close(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, filename); err != nil {
		return fmt.Errorf("replace config file: %w", err)
	}
	return nil
}
