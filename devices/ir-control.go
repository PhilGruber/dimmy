package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type IRControl struct {
	Device

	commands    map[string]string
	nextRequest *IrControlMessage
}

type IrControlMessage struct {
	IrCode string `json:"ir_code_to_send"`
}

func makeIrcontrol(config core.DeviceConfig) IRControl {
	i := IRControl{}
	i.Emoji = "📡"
	i.setBaseConfig(config)

	i.Type = "IRControl"
	i.commands = *config.Options.Commands

	log.Printf("IRControl device %s created with commands: %s\n", i.Name, i.GetCommands())

	return i
}

func NewIrControl(config core.DeviceConfig) *IRControl {
	i := makeIrcontrol(config)
	return &i
}

func (i *IRControl) ProcessRequest(request core.SwitchRequest) {
	log.Printf("Processing request for device %s: %v\n", i.Name, request.Value)
	command, ok := i.commands[request.Value]
	if !ok {
		log.Printf("Device %s does not support command %s. Please define this in config file.\n", i.Name, request.Value)
		return
	}
	req := IrControlMessage{IrCode: command}
	i.nextRequest = &req
}

func (i *IRControl) PublishValue(mqtt mqtt.Client) {
	if i.nextRequest == nil {
		return
	}
	s, _ := json.Marshal(i.nextRequest)
	mqtt.Publish(i.MqttTopic, 0, false, s)
	i.nextRequest = nil
}

func (i *IRControl) GetMax() int {
	return 1
}
func (i *IRControl) GetMin() int {
	return 1
}

func (i *IRControl) UpdateValue() (float64, bool) {
	return 0, i.nextRequest != nil
}

func (i *IRControl) GetCommands() []string {
	var commands []string
	for k, _ := range i.commands {
		commands = append(commands, k)
	}
	return commands
}
