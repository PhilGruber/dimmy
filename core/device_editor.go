package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var ErrDeviceNotFound = errors.New("configured device not found")

type DeviceEditorFields struct {
	Hidden  *bool
	Label   string
	Icon    string
	Min     *int
	Max     *int
	Devices *[]string
}

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
func UpdateDeviceFields(filename, name, deviceType, room, protocol string, fields ...DeviceEditorFields) error {
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
	if len(fields) > 0 {
		setField := func(mapping *yaml.Node, key string, value any) error {
			for i := 0; i+1 < len(mapping.Content); i += 2 {
				if mapping.Content[i].Value == key {
					if value == nil {
						mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
						return nil
					}
					return mapping.Content[i+1].Encode(value)
				}
			}
			if value == nil {
				return nil
			}
			var node yaml.Node
			if err := node.Encode(value); err != nil {
				return err
			}
			mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &node)
			return nil
		}
		if err := setField(target, "label", fields[0].Label); err != nil {
			return err
		}
		if err := setField(target, "icon", fields[0].Icon); err != nil {
			return err
		}
		var options *yaml.Node
		for i := 0; i+1 < len(target.Content); i += 2 {
			if target.Content[i].Value == "options" {
				options = target.Content[i+1]
				break
			}
		}
		if options == nil && (fields[0].Min != nil || fields[0].Max != nil || fields[0].Hidden != nil || fields[0].Devices != nil) {
			options = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			target.Content = append(target.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "options"}, options)
		}
		if options != nil {
			if options.Tag == "!!null" {
				*options = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
			if options.Kind != yaml.MappingNode {
				return errors.New("device options must be a YAML mapping")
			}
			if fields[0].Hidden != nil {
				if err := setField(options, "hidden", *fields[0].Hidden); err != nil {
					return err
				}
			}
			if fields[0].Devices != nil {
				if err := setField(options, "devices", *fields[0].Devices); err != nil {
					return err
				}
			}
			for _, bound := range []struct {
				key   string
				value *int
			}{{"min", fields[0].Min}, {"max", fields[0].Max}} {
				var value any
				if bound.value != nil {
					value = *bound.value
				}
				if err := setField(options, bound.key, value); err != nil {
					return err
				}
			}
		}
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
