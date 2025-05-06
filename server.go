package main

import (
	"encoding/json"
	"flag"
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

var AppVersion = "undefined"

type Server struct {
	devices map[string]dimmyDevices.DeviceInterface
	panels  map[string]dimmyDevices.Panel
	rules   []dimmyDevices.Rule
}

func main() {
	version := flag.Bool("version", false, "Print version")
	flag.Parse()

	if *version {
		fmt.Printf("dimmyd version %s\n", AppVersion)
		os.Exit(0)
	}

	server := &Server{}
	server.initialize()
}

func (s *Server) initialize() {

	s.devices = make(map[string]dimmyDevices.DeviceInterface)
	s.panels = make(map[string]dimmyDevices.Panel)

	config, err := core.LoadConfig()
	if err != nil {
		log.Println(err.Error())
		return
	}

	for _, deviceConfig := range config.Devices {
		switch deviceConfig.Type {
		case "motion-sensor":
		case "sensor":
			s.devices[deviceConfig.Name] = dimmyDevices.NewSensor(deviceConfig)
		case "zsensor":
			s.devices[deviceConfig.Name] = dimmyDevices.NewZSensor(deviceConfig)
		case "switch":
			s.devices[deviceConfig.Name] = dimmyDevices.NewSwitch(deviceConfig)
		case "door-sensor":
			s.devices[deviceConfig.Name] = dimmyDevices.NewDoorSensor(deviceConfig)
		case "light":
			s.devices[deviceConfig.Name] = dimmyDevices.NewLight(deviceConfig)
		case "zlight":
			s.devices[deviceConfig.Name] = dimmyDevices.NewZLight(deviceConfig)
		case "plug":
			s.devices[deviceConfig.Name] = dimmyDevices.NewPlug(deviceConfig)
		case "zplug":
			s.devices[deviceConfig.Name] = dimmyDevices.NewZPlug(deviceConfig)
		case "temperature":
			s.devices[deviceConfig.Name] = dimmyDevices.NewTemperature(deviceConfig)
		case "ircontrol":
			s.devices[deviceConfig.Name] = dimmyDevices.NewIrControl(deviceConfig)
		case "group":
		default:
			log.Println("Skipping deviceConfig of unknown type '" + deviceConfig.Type + "'")
		}
	}

	s.devices["time"] = dimmyDevices.NewDimmyTime(core.DeviceConfig{Name: "time", Type: "time"})

	// Parse Groups separately at the end, to make sure all referencing Devices exist at that point
	for _, device := range config.Devices {
		if device.Type == "group" {
			group := dimmyDevices.NewGroup(device, s.devices)
			if group != nil {
				s.devices[device.Name] = group
			}
		}
	}

	for _, ruleConfig := range config.Rules {
		rule := dimmyDevices.NewRule(ruleConfig, s.devices)
		if rule != nil {
			s.rules = append(s.rules, *rule)
		}
	}

	for _, panel := range config.Panels {
		s.panels[panel.Label] = dimmyDevices.NewPanel(panel, &s.devices)
	}

	for _, device := range s.devices {
		if !device.GetHidden() {
			s.panels[device.GetLabel()] = dimmyDevices.NewPanelFromDevice(device)
		}
	}

	channel := make(chan core.SwitchRequest, len(s.devices))

	go s.processRequests(channel)
	go s.eventLoop(channel, config.MqttServer)

	assets := http.FileServer(http.Dir(config.WebRoot + "/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", assets))
	http.Handle("/api/switch", ReceiveRequest(channel))
	http.Handle("/api/status", ShowStatus(&s.devices))
	http.Handle("/", ShowDashboard(s.devices, s.panels, channel, config.WebRoot))
	http.Handle("/rules/add-single-use", s.AddRule(config.WebRoot))

	log.Printf("Listening on port %d", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}

func (s *Server) eventLoop(channel chan core.SwitchRequest, mqttServer string) {
	hostname, _ := os.Hostname()
	client := initMqtt(mqttServer, "goserver-"+hostname)

	for name := range s.devices {
		if s.devices[name].GetMqttStateTopic() != "" {
			log.Printf("[%32s] Subscribing to %s\n", name, s.devices[name].GetMqttStateTopic())
			client.Subscribe(s.devices[name].GetMqttStateTopic(), 0, s.devices[name].GetMessageHandler(channel, s.devices[name]))
			s.devices[name].PollValue(client)
		}
	}

	for {

		for name := range s.devices {
			if _, ok := s.devices[name].UpdateValue(); ok {
				s.devices[name].PublishValue(client)
			}
		}

		var firedRules []int
		log.Printf("We currently have %d rules\n", len(s.rules))
		for idx, rule := range s.rules {
			//			fmt.Printf("Checking rule %s\n", rule.String())
			if rule.CheckTriggers() {
				//				fmt.Printf("\tFiring!\n", rule.String())
				rule.Fire(channel)
				firedRules = append(firedRules, idx)
			}
		}

		for _, idx := range firedRules {
			if s.rules[idx].SingleUse {
				// TODO: Make sure to remove the rule from all devices
				// TODO: Make sure this is legal
				s.rules = append(s.rules[:idx], s.rules[idx+1:]...)
				continue
			}
			s.rules[idx].ClearTriggers()
		}

		time.Sleep(core.CycleLength * time.Millisecond)
	}
}

func (s *Server) processRequests(channel chan core.SwitchRequest) {
	for {
		request := <-channel
		for _, device := range strings.Split(request.Device, ",") {
			if _, ok := s.devices[device]; ok {
				s.devices[device].ProcessRequest(request)
			} else {
				log.Printf("Can't find device for request [%s (%s)]", device, request.Device)
			}
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
		for _, device := range *devices {
			device.Lock()
		}
		jsonDevices, _ := json.Marshal(devices)
		for _, device := range *devices {
			device.Unlock()
		}
		_, _ = fmt.Fprintf(output, string(jsonDevices))
	}
}

func (s *Server) AddRule(webroot string) http.HandlerFunc {
	var devices []dimmyDevices.DeviceInterface
	for _, device := range s.devices {
		if device.HasReceivers() {
			devices = append(devices, device)
		}
	}
	return func(output http.ResponseWriter, request *http.Request) {
		if request.Method == "POST" {
			var form map[string]string
			err := json.NewDecoder(request.Body).Decode(&form)
			if err != nil {
				log.Println("Error: ", err)
				return
			}

			log.Printf("Form: %v\n", form)

			var dimmyTime *dimmyDevices.DimmyTime
			dimmyTime = s.devices["time"].(*dimmyDevices.DimmyTime)

			var unit time.Duration
			switch form["unit"] {
			case "seconds":
				unit = time.Second
			case "minutes":
				unit = time.Minute
			case "hours":
				unit = time.Hour
			}
			triggerTime := time.Now().Add(core.CycleLength * unit)

			ruleConfig := core.RuleConfig{
				Receivers: []core.ReceiverConfig{
					{
						DeviceName: form["device"],
						Key:        "command",
						Value:      form["value"],
					},
				},
				Triggers: dimmyTime.CreateTriggersFromTime(triggerTime),
			}

			log.Printf("Rec: %v\n", ruleConfig.Receivers[0])

			rule := dimmyDevices.NewRule(ruleConfig, s.devices)
			rule.SingleUse = true

			s.rules = append(s.rules, *rule)

			return
		}
		templ, err := template.ParseFiles(webroot + "/add-rule.html")
		if err != nil {
			log.Println(err)
			return
		}
		err = templ.Execute(output, struct {
			Devices []dimmyDevices.DeviceInterface
		}{devices})
		if err != nil {
			return
		}
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
