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
	"strconv"
	"strings"
	"time"
)

var AppVersion = "undefined"

type Server struct {
	devices map[string]dimmyDevices.DeviceInterface
	panels  map[string]dimmyDevices.Panel
	rules   []dimmyDevices.Rule
	channel chan core.SwitchRequest
}

func main() {
	version := flag.Bool("version", false, "Print version")
	flag.Parse()

	if *version {
		fmt.Printf("dimmyd version %s\n", AppVersion)
		os.Exit(0)
	}

	config, err := core.LoadConfig()
	if err != nil {
		log.Println(err.Error())
		return
	}

	server := &Server{}
	server.initialize(config)
	server.Start(config)
}

func (s *Server) initialize(config *core.ServerConfig) {

	s.devices = make(map[string]dimmyDevices.DeviceInterface)
	s.panels = make(map[string]dimmyDevices.Panel)

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
	s.devices["shell"] = dimmyDevices.NewShell(core.DeviceConfig{Name: "shell", Type: "shell"})

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

	s.channel = make(chan core.SwitchRequest, len(s.devices))
}

func (s *Server) Start(config *core.ServerConfig) {

	go s.processRequests()
	go s.eventLoop(config.MqttServer)

	assets := http.FileServer(http.Dir(config.WebRoot + "/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", assets))
	http.Handle("/api/switch", s.ReceiveRequest())
	http.Handle("/api/status", s.ShowStatus(&s.devices))
	http.Handle("/", s.ShowDashboard(config.WebRoot))
	http.Handle("/rules/add-single-use", s.AddSingleUseRule(config.WebRoot))

	log.Printf("Listening on port %d", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}

func (s *Server) eventLoop(mqttServer string) {
	hostname, _ := os.Hostname()
	client := s.initMqtt(mqttServer, "goserver-"+hostname)

	for name := range s.devices {
		if s.devices[name].GetMqttStateTopic() != "" {
			log.Printf("[%32s] Subscribing to %s\n", name, s.devices[name].GetMqttStateTopic())
			client.Subscribe(s.devices[name].GetMqttStateTopic(), 0, s.devices[name].GetMessageHandler(s.channel, s.devices[name]))
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
		for idx, rule := range s.rules {
			//			fmt.Printf("Checking rule %s\n", rule.String())
			if rule.CheckTriggers() {
				//				fmt.Printf("\tFiring!\n", rule.String())
				rule.Fire(s.channel)
				firedRules = append(firedRules, idx)
			}
		}

		for _, idx := range firedRules {
			if s.rules[idx].SingleUse {
				for _, trigger := range s.rules[idx].Triggers {
					trigger.Device.RemoveRule(&s.rules[idx])
				}
				s.rules = append(s.rules[:idx], s.rules[idx+1:]...)
				continue
			}
			s.rules[idx].ClearTriggers()
		}

		time.Sleep(core.CycleLength * time.Millisecond)
	}
}

func (s *Server) processRequests() {
	for {
		request := <-s.channel
		for _, device := range strings.Split(request.Device, ",") {
			if _, ok := s.devices[device]; ok {
				s.devices[device].ProcessRequest(request)
			} else {
				log.Printf("Can't find device for request [%s (%s)]", device, request.Device)
			}
		}
	}
}

func (s *Server) ReceiveRequest() http.HandlerFunc {
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
		s.channel <- request
		_, _ = fmt.Fprintf(output, jsonResponse(true, request, fmt.Sprintf("Sent %s command to %s", request.Command, request.Device)))
	}
}

func (s *Server) ShowStatus(devices *map[string]dimmyDevices.DeviceInterface) http.HandlerFunc {
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

func (s *Server) AddSingleUseRule(webroot string) http.HandlerFunc {
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
			in, err := strconv.Atoi(form["in"])
			if err != nil {
				log.Println("Error: ", err)
				return
			}
			triggerTime := time.Now().Add(unit * time.Duration(in))

			ruleConfig := core.RuleConfig{
				Receivers: []core.ReceiverConfig{
					{
						DeviceName: form["device"],
						Key:        form["key"],
						Value:      form["value"],
					},
				},
				Triggers: dimmyTime.CreateTriggersFromTime(triggerTime),
			}
			if s.devices[form["device"]].GetType() == "light" {
				ruleConfig.Receivers = append(ruleConfig.Receivers, core.ReceiverConfig{
					DeviceName: form["device"],
					Key:        "duration",
					Value:      "1",
				})
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

func (s *Server) ShowDashboard(webroot string) http.HandlerFunc {
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
					sr.Value = fmt.Sprintf("%.3f", s.devices[sr.Device].GetCurrent()+10)
				case "-":
					sr.Value = fmt.Sprintf("%.3f", s.devices[sr.Device].GetCurrent()-10)
				}
				s.channel <- sr
			}
		}

		templ, _ := template.ParseFiles(webroot + "/dashboard.html")
		err := templ.Execute(output, struct {
			Devices map[string]dimmyDevices.DeviceInterface
			Panels  map[string]dimmyDevices.Panel
		}{s.devices, s.panels})
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func (s *Server) initMqtt(hostname string, clientId string) mqtt.Client {
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
