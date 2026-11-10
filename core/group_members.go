package core

import "fmt"

// GroupMemberType matches the runtime device types used by NewGroup.
func GroupMemberType(deviceType string) string {
	switch deviceType {
	case "zlight":
		return "light"
	case "device":
		return "sensor"
	case "ircontrol":
		return "IRControl"
	}
	return deviceType
}

func ValidateGroupMembers(group string, members []string, devices []DeviceConfig) error {
	byName := make(map[string]DeviceConfig, len(devices))
	for _, device := range devices {
		byName[device.Name] = device
	}
	seen := make(map[string]bool)
	memberType := ""
	for _, name := range members {
		device, ok := byName[name]
		if !ok {
			return fmt.Errorf("group device %q was not found", name)
		}
		if name == group {
			return fmt.Errorf("a group cannot contain itself")
		}
		if device.Type == "group" {
			return fmt.Errorf("groups cannot be added as members")
		}
		if seen[name] {
			return fmt.Errorf("device %q is already in the group", name)
		}
		seen[name] = true
		currentType := GroupMemberType(device.Type)
		if memberType != "" && currentType != memberType {
			return fmt.Errorf("all group devices must have the same type (%s)", memberType)
		}
		memberType = currentType
	}
	return nil
}
