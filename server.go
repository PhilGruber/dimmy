package main

import (
	"encoding/json"
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {

	devices := make(map[string]dimmyDevices.DeviceInterface)
	panels := make(map[string]dimmyDevices.Panel)

	config, err := core.LoadConfig()
	if err != nil {
		log.Println(err.Error())
		return
	}

	for _, deviceConfig := range config.Devices {
		switch deviceConfig.Type {
		case "sensor":
			devices[deviceConfig.Name] = dimmyDevices.NewSensor(deviceConfig)
		case "zsensor":
			devices[deviceConfig.Name] = dimmyDevices.NewZSensor(deviceConfig)
		case "switch":
			devices[deviceConfig.Name] = dimmyDevices.NewSwitch(deviceConfig)
		case "door-sensor":
			devices[deviceConfig.Name] = dimmyDevices.NewDoorSensor(deviceConfig)
		case "light":
			devices[deviceConfig.Name] = dimmyDevices.NewLight(deviceConfig)
		case "zlight":
			devices[deviceConfig.Name] = dimmyDevices.NewZLight(deviceConfig)
		case "plug":
			devices[deviceConfig.Name] = dimmyDevices.NewPlug(deviceConfig)
		case "zplug":
			devices[deviceConfig.Name] = dimmyDevices.NewZPlug(deviceConfig)
		case "temperature":
			devices[deviceConfig.Name] = dimmyDevices.NewTemperature(deviceConfig)
		case "ztemperature":
			devices[deviceConfig.Name] = dimmyDevices.NewZTemperature(deviceConfig)
		case "ircontrol":
			devices[deviceConfig.Name] = dimmyDevices.NewIrControl(deviceConfig)
		case "group":
		default:
			log.Println("Skipping deviceConfig of unknown type '" + deviceConfig.Type + "'")
		}
	}

	devices["time"] = dimmyDevices.NewDimmyTime(core.DeviceConfig{Name: "time", Type: "time"})

	// Parse Groups separately at the end, to make sure all referencing Devices exist at that point
	for _, device := range config.Devices {
		if device.Type == "group" {
			group := dimmyDevices.NewGroup(device, devices)
			if group != nil {
				devices[device.Name] = group
			}
		}
	}

	var rules []dimmyDevices.Rule
	for _, ruleConfig := range config.Rules {
		rule := dimmyDevices.NewRule(ruleConfig, devices)
		if rule != nil {
			rules = append(rules, *rule)
		}
	}

	for _, panel := range config.Panels {
		panels[panel.Label] = dimmyDevices.NewPanel(panel, &devices)
	}

	for _, device := range devices {
		if !device.GetHidden() {
			panels[device.GetLabel()] = dimmyDevices.NewPanelFromDevice(device)
		}
	}

	channel := make(chan core.SwitchRequest, 10)

	go eventLoop(devices, rules, channel, config.MqttServer)

	assets := http.FileServer(http.Dir(config.WebRoot + "/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", assets))
	http.Handle("/api/switch", ReceiveRequest(channel))
	http.Handle("/api/status", ShowStatus(&devices))
	http.Handle("/", ShowDashboard(devices, panels, channel, config.WebRoot))

	log.Printf("Listening on port %d", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}

func eventLoop(devices map[string]dimmyDevices.DeviceInterface, rules []dimmyDevices.Rule, channel chan core.SwitchRequest, mqttServer string) {
	hostname, _ := os.Hostname()
	client := initMqtt(mqttServer, "goserver-"+hostname)

	for name := range devices {
		if devices[name].GetMqttStateTopic() != "" {
			log.Println("Subscribing to " + devices[name].GetMqttStateTopic())
			client.Subscribe(devices[name].GetMqttStateTopic(), 0, devices[name].GetMessageHandler(channel, devices[name]))
			devices[name].PollValue(client)
		} else {
			log.Println("No state topic for " + name)
		}
	}

	for {
		time.Sleep(core.CycleLength * time.Millisecond)

		for len(channel) > 0 {
			request := <-channel
			for _, device := range strings.Split(request.Device, ",") {
				if _, ok := devices[device]; ok {
					log.Println("Processing request for " + device)
					devices[device].ProcessRequest(request)
				} else {
					log.Printf("Can't find device for request [%s (%s)]", device, request.Device)
				}
			}
		}

		for name := range devices {
			if _, ok := devices[name].UpdateValue(); ok {
				devices[name].PublishValue(client)
			}
		}

		var firedRules []dimmyDevices.Rule
		for _, rule := range rules {
			if rule.CheckTriggers() {
				rule.Fire(channel)
				firedRules = append(firedRules, rule)
			}
		}

		for _, rule := range firedRules {
			rule.ClearTriggers()
		}
	}
}

func ReceiveRequest(channel chan core.SwitchRequest) http.HandlerFunc {
	return func(output http.ResponseWriter, httpRequest *http.Request) {

		jsonResponse := func(result bool, request interface{}, message string) string {
			data := make(map[string]interface{})
			data["data"] = "Success"
			data["input"] = request
			data["message"] = message
			jsonData, _ := json.Marshal(data)
			return string(jsonData)
		}

		body, err := io.ReadAll(httpRequest.Body)
		if err != nil {
			log.Println("Error: ", err)
			_, _ = fmt.Fprintf(output, "Invalid JSON data")
			return
		}
		log.Println("Received payload from api: " + string(body[:]))

		var request core.SwitchRequest

		err = json.Unmarshal(body, &request)

		if err != nil {
			log.Println("Error: ", err)
			log.Println(string(body[:]))
			_, _ = fmt.Fprintf(output, "Invalid JSON data")
			return
		}
		channel <- request
		_, _ = fmt.Fprintf(output, jsonResponse(true, request, fmt.Sprintf("Sent %s command to %s", request.Command, request.Device)))
	}
}

func ShowStatus(devices *map[string]dimmyDevices.DeviceInterface) http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		output.Header().Set("Content-Type", "application/json")
		jsonDevices, _ := json.Marshal(devices)
		_, _ = fmt.Fprintf(output, string(jsonDevices))
	}
}

func ShowDashboard(devices map[string]dimmyDevices.DeviceInterface, panels map[string]dimmyDevices.Panel, channel chan core.SwitchRequest, webroot string) http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		if request.Method == "POST" {
			err := request.ParseForm()
			if err == nil {
				var sr core.SwitchRequest
				sr.Duration = 0
				sr.Device = request.FormValue("device")
				target := request.FormValue("target")
				switch target {
				case "on":
					sr.Value = "100"
				case "off":
					sr.Value = "0"
				case "+":
					sr.Value = fmt.Sprintf("%.3f", devices[sr.Device].GetCurrent()+10)
				case "-":
					sr.Value = fmt.Sprintf("%.3f", devices[sr.Device].GetCurrent()-10)
				}
				channel <- sr
			}
		}

		templ, _ := template.ParseFiles(webroot + "/dashboard.html")
		err := templ.Execute(output, struct {
			Devices map[string]dimmyDevices.DeviceInterface
			Panels  map[string]dimmyDevices.Panel
		}{devices, panels})
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func initMqtt(hostname string, clientId string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:1883", hostname))
	opts.SetClientID(clientId)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(5 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to MQTT at " + hostname)
	return client
}
