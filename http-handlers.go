package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
)

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
		request.Force = true

		if err != nil {
			log.Println("Error: ", err)
			log.Println(string(body[:]))
			_, _ = fmt.Fprintf(output, "Invalid JSON data")
			return
		}
		s.channel <- request
		_, _ = io.WriteString(output, jsonResponse(true, request, fmt.Sprintf("Sent %s command to %s", request.Command, request.Device)))
	}
}

func (s *Server) ShowStatus(devices *map[string]dimmyDevices.DeviceInterface) http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		output.Header().Set("Content-Type", "application/json")
		snapshot := s.deviceSnapshot()
		for _, device := range snapshot {
			device.Lock()
		}
		jsonDevices, _ := json.Marshal(snapshot)
		for _, device := range snapshot {
			device.Unlock()
		}
		_, _ = output.Write(jsonDevices)
	}
}

type unknownDeviceView struct {
	Name     string
	Topic    string
	Type     string
	Sensors  []string
	Controls []string
}

func (s *Server) ShowUnknownDevices(webroot string) http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		s.mutex.RLock()
		devices := make([]unknownDeviceView, 0, len(s.unknownDevices))
		for _, device := range s.unknownDevices {
			view := unknownDeviceView{
				Name:  device.GetName(),
				Topic: device.GetMqttTopic(),
				Type:  device.GetType(),
			}
			if generic, ok := device.(*dimmyDevices.GenericDevice); ok {
				for _, sensor := range generic.GetSensors() {
					view.Sensors = append(view.Sensors, sensor.Name)
				}
				for _, control := range generic.GetControls() {
					view.Controls = append(view.Controls, control.Name)
				}
			}
			devices = append(devices, view)
		}
		s.mutex.RUnlock()
		sort.Slice(devices, func(i, j int) bool {
			return devices[i].Topic < devices[j].Topic
		})

		templ, err := template.ParseFiles(webroot + "/unknown-devices.html")
		if err != nil {
			http.Error(output, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := templ.Execute(output, struct {
			Devices []unknownDeviceView
		}{devices}); err != nil {
			log.Println(err)
		}
	}
}

func (s *Server) SaveUnknownDevice() http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(output, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := request.ParseForm(); err != nil {
			http.Error(output, "invalid form data", http.StatusBadRequest)
			return
		}

		topic := strings.TrimSpace(request.FormValue("topic"))
		name := strings.TrimSpace(request.FormValue("name"))
		if topic == "" || name == "" {
			http.Error(output, "topic and device name are required", http.StatusBadRequest)
			return
		}

		s.mutex.Lock()

		if _, exists := s.devices[name]; exists {
			s.mutex.Unlock()
			http.Error(output, "a device with that name already exists", http.StatusConflict)
			return
		}
		device, exists := s.unknownDevices[topic]
		if !exists {
			s.mutex.Unlock()
			http.Error(output, "unknown device was not found", http.StatusNotFound)
			return
		}
		generic, ok := device.(*dimmyDevices.GenericDevice)
		if !ok {
			s.mutex.Unlock()
			http.Error(output, "unknown device type cannot be saved", http.StatusUnprocessableEntity)
			return
		}

		deviceConfig := generic.Config(name)
		if err := core.AddDeviceToConfig(s.config.Filename, deviceConfig); err != nil {
			s.mutex.Unlock()
			log.Printf("Could not save device %s: %s", topic, err)
			http.Error(output, "could not update config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		generic.Name = name
		generic.Label = name
		s.devices[name] = generic
		delete(s.unknownDevices, topic)
		s.config.Devices = append(s.config.Devices, deviceConfig)
		mqttClient := s.mqttClient
		s.mutex.Unlock()

		if mqttClient != nil && generic.GetMqttStateTopic() != "" {
			token := mqttClient.Subscribe(generic.GetMqttStateTopic(), 0, generic.GetMessageHandler(s.channel, generic))
			if token.Wait() && token.Error() != nil {
				log.Printf("Could not subscribe saved device %s: %s", name, token.Error())
			}
			generic.PollValue(mqttClient)
		}

		output.Header().Set("Content-Type", "application/json")
		output.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(output, `{"name":%q}`, name)
	}
}

func (s *Server) AddSingleUseRule(webroot string) http.HandlerFunc {
	var devices []dimmyDevices.DeviceInterface
	for _, device := range s.deviceSnapshot() {
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
			timeDevice, ok := s.getDevice("time")
			if !ok {
				http.Error(output, "time device is unavailable", http.StatusInternalServerError)
				return
			}
			dimmyTime = timeDevice.(*dimmyDevices.DimmyTime)

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
			targetDevice, ok := s.getDevice(form["device"])
			if !ok {
				http.Error(output, "device was not found", http.StatusNotFound)
				return
			}
			if targetDevice.GetType() == "light" {
				ruleConfig.Receivers = append(ruleConfig.Receivers, core.ReceiverConfig{
					DeviceName: form["device"],
					Key:        "duration",
					Value:      "1",
				})
			}

			log.Printf("Rec: %v\n", ruleConfig.Receivers[0])

			rule := dimmyDevices.NewRule(ruleConfig, s.deviceSnapshot())
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

func (s *Server) ShowDashboard(webroot string, name string) http.HandlerFunc {
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
					if device, ok := s.getDevice(sr.Device); ok {
						sr.Value = fmt.Sprintf("%.3f", device.GetCurrent()+10)
					}
				case "-":
					if device, ok := s.getDevice(sr.Device); ok {
						sr.Value = fmt.Sprintf("%.3f", device.GetCurrent()-10)
					}
				}
				s.channel <- sr
			}
		}

		templ, err := template.ParseFiles(webroot + "/dashboard.html")
		if err != nil {
			log.Println(err)
			return
		}
		err = templ.Execute(output, struct {
			Devices map[string]dimmyDevices.DeviceInterface
			Panels  []dimmyDevices.Panel
		}{s.deviceSnapshot(), s.dashboards[name]})
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func (s *Server) EditRules(webroot string) http.HandlerFunc {
	return func(output http.ResponseWriter, httpRequest *http.Request) {
		templ, _ := template.ParseFiles(webroot + "/rules.html")
		err := templ.Execute(output, struct {
			Rules []dimmyDevices.Rule
		}{s.rules})
		if err != nil {
			log.Println(err)
			return
		}
	}
}
