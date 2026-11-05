package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var ErrDeviceNotFound = errors.New("configured device not found")

func LoadConfiguredDevices(filename string) ([]DeviceConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config struct {
		Devices []DeviceConfig `yaml:"devices"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config.Devices, nil
}

// UpdateDeviceFields changes only the fields exposed by the shared device form.
func UpdateDeviceFields(filename, name, deviceType, room, protocol string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return errors.New("config file must contain a YAML mapping")
	}
	root := doc.Content[0]
	var target *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "devices" {
			continue
		}
		if root.Content[i+1].Kind != yaml.SequenceNode {
			return errors.New("config devices must be a YAML sequence")
		}
		for _, node := range root.Content[i+1].Content {
			if node.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j+1 < len(node.Content); j += 2 {
				if node.Content[j].Value == "name" && node.Content[j+1].Value == name {
					if target != nil {
						return fmt.Errorf("multiple devices named %s", name)
					}
					target = node
				}
			}
		}
	}
	if target == nil {
		return ErrDeviceNotFound
	}
	for _, field := range []struct{ key, value string }{{"type", deviceType}, {"room", room}, {"protocol", protocol}} {
		var valueNode *yaml.Node
		for i := 0; i+1 < len(target.Content); i += 2 {
			if target.Content[i].Value == field.key {
				valueNode = target.Content[i+1]
				break
			}
		}
		if valueNode == nil {
			valueNode = &yaml.Node{}
			target.Content = append(target.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: field.key}, valueNode)
		}
		valueNode.Kind, valueNode.Tag, valueNode.Value = yaml.ScalarNode, "!!str", field.value
	}
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(filename), ".device-editor-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	encoder := yaml.NewEncoder(temp)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), filename)
}
