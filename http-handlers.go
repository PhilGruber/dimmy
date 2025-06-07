package main

import (
	"encoding/json"
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
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
		if device.HasReceivers() && !device.GetHidden() {
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
			Panels  []dimmyDevices.Panel
		}{s.devices, s.panels})
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
