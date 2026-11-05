package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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

		body, err := io.ReadAll(http.MaxBytesReader(output, httpRequest.Body, maxRequestBodyBytes))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(output, "request body too large", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(output, "could not read request body", http.StatusBadRequest)
			}
			return
		}

		var request core.SwitchRequest

		err = json.Unmarshal(body, &request)
		request.Force = true

		if err != nil {
			http.Error(output, "invalid JSON data", http.StatusBadRequest)
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
	Protocol string
	Name     string
	Topic    string
	Type     string
	Sensors  []string
	Controls []string
}

func (s *Server) EditDevices(webroot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.mutex.RLock()
		filename := s.config.Filename
		s.mutex.RUnlock()
		devices, err := core.LoadConfiguredDevices(filename)
		if err != nil {
			http.Error(w, "could not load device configuration", http.StatusInternalServerError)
			return
		}
		rooms := make([]string, 0)
		seen := make(map[string]bool)
		for _, device := range devices {
			if device.Room != "" && !seen[device.Room] {
				rooms = append(rooms, device.Room)
				seen[device.Room] = true
			}
		}
		sort.Strings(rooms)
		sort.SliceStable(devices, func(i, j int) bool { return devices[i].Name < devices[j].Name })
		type roomGroup struct {
			Room    string
			Devices []core.DeviceConfig
		}
		byRoom := make(map[string][]core.DeviceConfig)
		for _, device := range devices {
			room := strings.TrimSpace(device.Room)
			byRoom[room] = append(byRoom[room], device)
		}
		groupRooms := make([]string, 0, len(byRoom))
		for room := range byRoom {
			if room != "" {
				groupRooms = append(groupRooms, room)
			}
		}
		sort.Strings(groupRooms)
		if len(byRoom[""]) > 0 {
			groupRooms = append(groupRooms, "")
		}
		groups := make([]roomGroup, 0, len(groupRooms))
		for _, room := range groupRooms {
			groups = append(groups, roomGroup{Room: room, Devices: byRoom[room]})
		}
		data, err := json.Marshal(devices)
		if err != nil {
			http.Error(w, "could not encode devices", http.StatusInternalServerError)
			return
		}
		page := "device-editor.html"
		if r.URL.Path == "/devices" {
			page = "devices.html"
		}
		tmpl, err := template.New(page).Funcs(template.FuncMap{
			"iconURL": func(icon string) string {
				if icon != "" && filepath.Base(icon) == icon {
					if info, err := os.Stat(filepath.Join(webroot, "assets", "icons", icon)); err == nil && info.Mode().IsRegular() {
						return "/assets/icons/" + url.PathEscape(icon)
					}
				}
				return ""
			},
		}).ParseFiles(filepath.Join(webroot, page), webroot+"/device-form.html")
		if err != nil {
			http.Error(w, "could not load device editor", http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, struct {
			Devices      []core.DeviceConfig
			DeviceGroups []roomGroup
			Rooms        []string
			DevicesJSON  template.JS
			Saved        bool
		}{devices, groups, rooms, template.JS(data), r.URL.Query().Get("saved") == "1"}); err != nil {
			log.Println(err)
		}
	}
}

func (s *Server) SaveDeviceEdits() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}
		name := r.FormValue("name")
		deviceType := r.FormValue("type")
		protocol := r.FormValue("protocol")
		s.mutex.Lock()
		defer s.mutex.Unlock()
		devices, err := core.LoadConfiguredDevices(s.config.Filename)
		if err != nil {
			http.Error(w, "could not load device configuration", http.StatusInternalServerError)
			return
		}
		var original *core.DeviceConfig
		for i := range devices {
			if devices[i].Name == name {
				original = &devices[i]
				break
			}
		}
		if original == nil {
			http.Error(w, "configured device not found", http.StatusNotFound)
			return
		}
		if deviceType != original.Type {
			switch deviceType {
			case "device", "ircontrol", "light", "zlight":
			default:
				http.Error(w, "unsupported device type", http.StatusBadRequest)
				return
			}
		}
		if protocol != original.Protocol {
			switch protocol {
			case "", "zigbee", "cbus", "tasmota":
			default:
				http.Error(w, "unsupported device protocol", http.StatusBadRequest)
				return
			}
		}
		if err := core.UpdateDeviceFields(s.config.Filename, name, deviceType, strings.TrimSpace(r.FormValue("room")), protocol); err != nil {
			http.Error(w, "could not save device configuration", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Saved. Restart dimmyd to apply the changes."})
	}
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
			view.Protocol = "zigbee"
			prefix := strings.ToLower(view.Topic)
			if strings.HasPrefix(prefix, "cbus") {
				view.Protocol = "cbus"
			} else if strings.HasPrefix(prefix, "tele") || strings.HasPrefix(prefix, "stat") {
				view.Protocol = "tasmota"
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
		roomSet := make(map[string]bool)
		for _, device := range s.devices {
			if room := device.GetRoom(); room != "" {
				roomSet[room] = true
			}
		}
		s.mutex.RUnlock()
		rooms := make([]string, 0, len(roomSet))
		for room := range roomSet {
			rooms = append(rooms, room)
		}
		sort.Strings(rooms)
		sort.Slice(devices, func(i, j int) bool {
			return devices[i].Topic < devices[j].Topic
		})

		templ, err := template.ParseFiles(webroot+"/unknown-devices.html", webroot+"/device-form.html")
		if err != nil {
			http.Error(output, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := templ.Execute(output, struct {
			Devices []unknownDeviceView
			Rooms   []string
		}{Devices: devices, Rooms: rooms}); err != nil {
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
		deviceType := strings.TrimSpace(request.FormValue("type"))
		if topic == "" || name == "" {
			http.Error(output, "topic and device name are required", http.StatusBadRequest)
			return
		}

		room := strings.TrimSpace(request.FormValue("room"))
		protocol := strings.TrimSpace(request.FormValue("protocol"))
		switch protocol {
		case "", "zigbee", "cbus", "tasmota":
		default:
			http.Error(output, "unsupported device protocol", http.StatusBadRequest)
			return
		}

		log.Printf("Saving device %s of type %s (topic=%s)\n", name, deviceType, topic)

		var device dimmyDevices.DeviceInterface
		var mqttClient mqtt.Client
		saved := func() bool {
			s.mutex.Lock()
			defer s.mutex.Unlock()

			if _, exists := s.devices[name]; exists {
				http.Error(output, "a device with that name already exists", http.StatusConflict)
				return false
			}
			bareDevice, exists := s.unknownDevices[topic]
			if !exists {
				http.Error(output, "unknown device was not found", http.StatusNotFound)
				return false
			}
			deviceConfig := bareDevice.GetConfig(name)
			deviceConfig.Room = room
			if protocol != "" {
				deviceConfig.Protocol = protocol
			}
			switch deviceType {
			case "generic-device":
				device = s.newGenericDevice(deviceConfig)
			case "zlight":
				device = dimmyDevices.NewZLight(deviceConfig)
			case "light":
				device = dimmyDevices.NewLight(deviceConfig)
			case "ir-control":
				device = dimmyDevices.NewIrControl(deviceConfig)
			default:
				http.Error(output, "unknown device of type "+deviceType+" cannot be saved", http.StatusUnprocessableEntity)
				return false
			}
			deviceConfig = device.GetConfig(name)

			if err := core.AddDeviceToConfig(s.config.Filename, deviceConfig); err != nil {
				log.Printf("Could not save device %s: %s", topic, err)
				http.Error(output, "could not update config: "+err.Error(), http.StatusInternalServerError)
				return false
			}

			device.SetName(name)
			device.SetLabel(name)
			s.devices[name] = device
			delete(s.unknownDevices, topic)
			s.config.Devices = append(s.config.Devices, deviceConfig)
			mqttClient = s.mqttClient

			return true
		}()
		if !saved {
			return
		}

		if mqttClient != nil && device.GetMqttStateTopic() != "" {
			token := mqttClient.Subscribe(device.GetMqttStateTopic(), 0, device.GetMessageHandler(s.channel, device))
			if token.Wait() && token.Error() != nil {
				log.Printf("Could not subscribe saved device %s: %s", name, token.Error())
			}
			device.PollValue(mqttClient)
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
			dimmyTime, ok = timeDevice.(*dimmyDevices.DimmyTime)
			if !ok || dimmyTime == nil {
				http.Error(output, "time device is unavailable", http.StatusInternalServerError)
				return
			}

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

			s.rules = append(s.rules, rule)

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
		name := request.PathValue("dashboard")
		if name == "" {
			name = "default"
		}
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

func (s *Server) ShowSensorHistory(webroot string) http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		templ, err := template.ParseFiles(webroot + "/sensor-history.html")
		if err != nil {
			http.Error(output, "could not load history page: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := templ.Execute(output, nil); err != nil {
			log.Println(err)
		}
	}
}

func (s *Server) EditRules(webroot string) http.HandlerFunc {
	return func(output http.ResponseWriter, httpRequest *http.Request) {
		deviceView := func(device dimmyDevices.DeviceInterface) ruleDeviceView {
			view := ruleDeviceView{Name: device.GetName(), Label: device.GetLabel(), Icon: device.GetEmoji(),
				Triggers: nonNilStrings(device.GetTriggers()), Receivers: nonNilStrings(device.GetReceivers())}
			if view.Icon != "" && filepath.Base(view.Icon) == view.Icon {
				if info, err := os.Stat(filepath.Join(webroot, "assets", "icons", view.Icon)); err == nil && info.Mode().IsRegular() {
					view.IconURL = "/assets/icons/" + url.PathEscape(view.Icon)
				}
			}
			return view
		}
		templ, err := template.New("rules.html").Funcs(template.FuncMap{
			"deviceView": func(name string) ruleDeviceView {
				if device, ok := s.getDevice(name); ok {
					return deviceView(device)
				}
				return ruleDeviceView{Name: name, Label: name + " (unavailable)"}
			},
		}).ParseFiles(webroot + "/rules.html")
		if err != nil {
			http.Error(output, "could not load rules page", http.StatusInternalServerError)
			return
		}
		devices := make([]ruleDeviceView, 0)
		for _, device := range s.deviceSnapshot() {
			devices = append(devices, deviceView(device))
		}
		sort.Slice(devices, func(i, j int) bool { return devices[i].Label < devices[j].Label })
		s.mutex.RLock()
		configuredRules := append([]core.RuleConfig(nil), s.config.Rules...)
		s.mutex.RUnlock()
		rulesJSON, err := json.Marshal(configuredRules)
		if err != nil {
			http.Error(output, "could not encode rules", http.StatusInternalServerError)
			return
		}
		devicesJSON, err := json.Marshal(devices)
		if err != nil {
			http.Error(output, "could not encode devices", http.StatusInternalServerError)
			return
		}
		assetsHash := sha256.New()
		for _, asset := range []string{"rules.js", "rules.css"} {
			if data, err := os.ReadFile(filepath.Join(webroot, "assets", asset)); err == nil {
				assetsHash.Write(data)
			}
		}
		err = templ.Execute(output, struct {
			RuleGroups    []ruleRoomGroup
			RulesJSON     template.JS
			DevicesJSON   template.JS
			AssetsVersion string
		}{RuleGroups: groupRulesByRoom(configuredRules, s.deviceSnapshot()), RulesJSON: template.JS(rulesJSON), DevicesJSON: template.JS(devicesJSON), AssetsVersion: fmt.Sprintf("%x", assetsHash.Sum(nil))})
		if err != nil {
			log.Println(err)
			return
		}
	}
}

type indexedRule struct {
	core.RuleConfig
	Index int
}

type ruleRoomGroup struct {
	Room  string
	Rules []indexedRule
}

func groupRulesByRoom(rules []core.RuleConfig, devices map[string]dimmyDevices.DeviceInterface) []ruleRoomGroup {
	byRoom := make(map[string][]indexedRule)
	for index, rule := range rules {
		rooms := make(map[string]bool)
		addRoom := func(name string) {
			if device, ok := devices[name]; ok {
				if room := strings.TrimSpace(device.GetRoom()); room != "" {
					rooms[room] = true
				}
			}
		}
		for _, trigger := range rule.Triggers {
			addRoom(trigger.DeviceName)
		}
		for _, receiver := range rule.Receivers {
			addRoom(receiver.DeviceName)
		}
		room := "General"
		if len(rooms) > 1 {
			room = "Multiple Rooms"
		} else {
			for name := range rooms {
				room = name
			}
		}
		byRoom[room] = append(byRoom[room], indexedRule{RuleConfig: rule, Index: index})
	}
	groups := make([]ruleRoomGroup, 0, len(byRoom))
	for room, rules := range byRoom {
		groups = append(groups, ruleRoomGroup{Room: room, Rules: rules})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Room < groups[j].Room })
	return groups
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

type ruleDeviceView struct {
	Name      string   `json:"name"`
	Label     string   `json:"label"`
	Icon      string   `json:"icon"`
	IconURL   string   `json:"icon_url,omitempty"`
	Triggers  []string `json:"triggers"`
	Receivers []string `json:"receivers"`
}

func (s *Server) SaveRules() http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut {
			http.Error(output, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var rules []core.RuleConfig
		if err := json.NewDecoder(http.MaxBytesReader(output, request.Body, 1<<20)).Decode(&rules); err != nil {
			http.Error(output, "invalid rules: "+err.Error(), http.StatusBadRequest)
			return
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()
		devices := s.devices
		// Existing rules can be retained unchanged while other broken rules are
		// repaired or deleted. New and modified rules still require valid fields.
		retained := make([]bool, len(s.config.Rules))
		for _, rule := range rules {
			unchanged := false
			ruleJSON, _ := json.Marshal(rule)
			for i, existing := range s.config.Rules {
				existingJSON, err := json.Marshal(existing)
				if !retained[i] && err == nil && bytes.Equal(ruleJSON, existingJSON) {
					retained[i] = true
					unchanged = true
					break
				}
			}
			if unchanged {
				continue
			}
			if len(rule.Triggers) == 0 || len(rule.Receivers) == 0 {
				http.Error(output, "each rule needs a sensor and a control", http.StatusBadRequest)
				return
			}
			for _, trigger := range rule.Triggers {
				device, ok := devices[trigger.DeviceName]
				if !ok || !contains(device.GetTriggers(), trigger.Key) {
					http.Error(output, "invalid sensor: "+trigger.Key, http.StatusBadRequest)
					return
				}
			}
			for _, receiver := range rule.Receivers {
				device, ok := devices[receiver.DeviceName]
				if !ok || !contains(device.GetReceivers(), receiver.Key) {
					http.Error(output, "invalid control", http.StatusBadRequest)
					return
				}
			}
		}

		if err := core.SaveRulesConfig(s.config.RulesFilename, rules); err != nil {
			http.Error(output, "could not save rules: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, rule := range s.rules {
			for _, trigger := range rule.Triggers {
				trigger.Device.RemoveRule(rule)
			}
		}
		s.rules = nil
		s.config.Rules = rules
		for _, ruleConfig := range rules {
			s.rules = append(s.rules, dimmyDevices.NewRule(ruleConfig, s.devices))
		}
		output.Header().Set("Content-Type", "application/json")
		_, _ = output.Write([]byte(`{"ok":true}`))
	}
}

func (s *Server) LoadSensorHistory(db *core.HistoryDatabase) http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		device := request.PathValue("device")
		sensor := request.PathValue("sensor")
		result, err := db.GetSensorHistory(device, sensor)
		if err != nil {
			http.Error(output, "could not load sensor history: "+err.Error(), http.StatusInternalServerError)
			return
		}
		output.Header().Set("Content-Type", "application/json")
		blob, err := json.Marshal(result)
		if err != nil {
			http.Error(output, "could not load sensor history: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = output.Write(blob)
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
