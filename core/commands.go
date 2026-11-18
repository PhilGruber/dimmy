package core

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Command struct {
	Name  string
	Value string
}

// Commands retains the order of entries in the YAML commands mapping.
type Commands []Command

func (c *Commands) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("commands must be a YAML mapping")
	}
	// Decode once to validate mapping keys, values and duplicate names.
	var values map[string]string
	if err := node.Decode(&values); err != nil {
		return err
	}
	commands := make(Commands, 0, len(values))
	for i := 0; i+1 < len(node.Content); i += 2 {
		var name string
		if err := node.Content[i].Decode(&name); err != nil {
			return err
		}
		commands = append(commands, Command{Name: name, Value: values[name]})
	}
	*c = commands
	return nil
}

func (c Commands) MarshalYAML() (any, error) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, command := range c {
		var name, value yaml.Node
		if err := name.Encode(command.Name); err != nil {
			return nil, err
		}
		if err := value.Encode(command.Value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, &name, &value)
	}
	return node, nil
}

func (c Commands) Get(name string) (string, bool) {
	for _, command := range c {
		if command.Name == name {
			return command.Value, true
		}
	}
	return "", false
}

func (c Commands) Names() []string {
	names := make([]string, 0, len(c))
	for _, command := range c {
		names = append(names, command.Name)
	}
	return names
}
