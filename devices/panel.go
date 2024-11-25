package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"log"
)

type Panel struct {
	Label   string
	Devices []DeviceInterface
}

func NewPanel(config core.PanelConfig, devices *map[string]DeviceInterface) Panel {
	p := Panel{}
	p.Label = config.Label
	for _, dn := range config.Devices {
		for _, d := range *devices {
			if d.GetName() == dn {
				p.Devices = append(p.Devices, d)
				break
			}
		}
	}
	log.Printf("Created panel %s with %d devices\n", p.Label, len(p.Devices))
	return p
}

func NewPanelFromDevice(device DeviceInterface) Panel {
	p := Panel{}
	p.Label = device.GetLabel()
	p.Devices = append(p.Devices, device)
	log.Println("Created panel from Device " + device.GetName())
	return p
}

func (p Panel) GetLabel() string {
	return p.Label
}

func (p Panel) GetTemperatureDevice() *DeviceInterface {
	for _, d := range p.Devices {
		if d.GetType() == "temperature" {
			return &d
		}
	}
	return nil
}

func (p Panel) HasTemperatureDevice() bool {
	return p.GetTemperatureDevice() != nil
}

func (p Panel) GetDevices() []DeviceInterface {
	return p.Devices
}
