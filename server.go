package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v3"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {

	devices := make(map[string]dimmyDevices.DeviceInterface)

	config, err := loadConfig()
	if err != nil {
		log.Println(err.Error())
		return
	}

	log.Println("MQTT: " + config.MqttServer)

	for _, device := range config.Devices {
		switch device.Type {
		case "sensor":
			devices[device.Name] = dimmyDevices.NewSensor(device)
		case "zsensor":
			devices[device.Name] = dimmyDevices.NewZSensor(device)
		case "switch":
			devices[device.Name] = dimmyDevices.NewSwitch(device)
		case "light":
			devices[device.Name] = dimmyDevices.NewLight(device)
		case "zlight":
			devices[device.Name] = dimmyDevices.NewZLight(device)
		case "plug":
			devices[device.Name] = dimmyDevices.NewPlug(device)
		case "thermostat":
			devices[device.Name] = dimmyDevices.NewThermostat(device)
		case "group":
		default:
			log.Println("Skipping device of unknown type '" + device.Type + "'")
		}
	}

	// Parse Groups separately at the end, to make sure all referencing Devices exist at that point
	for _, device := range config.Devices {
		if device.Type == "group" {
			group := dimmyDevices.NewGroup(device, devices)
			if group != nil {
				devices[device.Name] = group
			}
		}
	}

	channel := make(chan core.SwitchRequest, 10)

	go eventLoop(devices, channel, config.MqttServer)

	assets := http.FileServer(http.Dir(config.WebRoot + "/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", assets))
	http.Handle("/api/switch", ReceiveRequest(channel))
	http.Handle("/api/status", ShowStatus(&devices))
	http.Handle("/", ShowDashboard(devices, channel, config.WebRoot))

	log.Printf("Listening on port %d", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}

func eventLoop(devices map[string]dimmyDevices.DeviceInterface, channel chan core.SwitchRequest, mqttServer string) {
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
					log.Printf("Unknown device [%s (%s)]", device, request.Device)
				}
			}
		}

		for name, _ := range devices {
			if _, ok := devices[name].UpdateValue(); ok {
				devices[name].PublishValue(client)
			}
		}

		for name, _ := range devices {
			if devices[name].GetType() == "sensor" {
				request, ok := devices[name].GetTimeoutRequest()
				if ok {
					channel <- request
				}

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

		body, err := ioutil.ReadAll(httpRequest.Body)
		if err != nil {
			log.Println("Error: ", err)
			fmt.Fprintf(output, "Invalid JSON data")
			return
		}

		var request core.SwitchRequest

		err = json.Unmarshal(body, &request)

		if err != nil {
			log.Println("Error: ", err)
			log.Println(string(body[:]))
			fmt.Fprintf(output, "Invalid JSON data")
			return
		}
		channel <- request
		fmt.Fprintf(output, jsonResponse(true, request, fmt.Sprintf("Sent %s command to %s", request.Command, request.Device)))
	}
}

func ShowStatus(devices *map[string]dimmyDevices.DeviceInterface) http.HandlerFunc {
	return func(output http.ResponseWriter, request *http.Request) {
		jsonDevices, _ := json.Marshal(devices)
		fmt.Fprintf(output, string(jsonDevices))
	}
}

func ShowDashboard(devices map[string]dimmyDevices.DeviceInterface, channel chan core.SwitchRequest, webroot string) http.HandlerFunc {
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
					sr.Value = float64(devices[sr.Device].GetMax())
				case "off":
					sr.Value = float64(devices[sr.Device].GetMin())
				case "+":
					sr.Value = devices[sr.Device].GetCurrent() + 10
				case "-":
					sr.Value = devices[sr.Device].GetCurrent() - 10
				}
				channel <- sr
			}
		}

		templ, _ := template.ParseFiles(webroot + "/dashboard.html")
		err := templ.Execute(output, devices)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func loadConfig() (*core.ServerConfig, error) {

	var filename string
	if _, err := os.Stat("/etc/dimmyd.conf.yaml"); err == nil {
		filename = "/etc/dimmyd.conf.yaml"
	} else if _, err := os.Stat("dimmyd.conf.yaml"); err == nil {
		filename = "dimmyd.conf.yaml"
	} else {
		return nil, errors.New("could not find config file /etc/dimmyd.conf.yaml")
	}

	log.Println("Loading config file " + filename)

	var config core.ServerConfig
	configYaml, _ := os.ReadFile(filename)
	err := yaml.Unmarshal(configYaml, &config)
	if err != nil {
		log.Fatal(err)
	}
	return &config, nil
}

func loadLegacyConfig() (map[string]string, map[string]map[string]string, error) {
	config := map[string]map[string]string{}

	var filename string

	if _, err := os.Stat("/etc/dimmyd.conf"); err == nil {
		filename = "/etc/dimmyd.conf"
	} else if _, err := os.Stat("dimmyd.conf"); err == nil {
		filename = "dimmyd.conf"
	} else {
		return nil, nil, errors.New("Could not find config file /etc/dimmyd.conf")
	}

	log.Println("Loading config file " + filename)

	file, err := os.Open(filename)

	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	var line string
	deviceName := "__global"
	config[deviceName] = map[string]string{}
	for {
		line, err = reader.ReadString('\n')
		line = strings.TrimSpace(line)

		if len(line) < 3 {
			if err != nil {
				break
			}
			continue
		}

		if line[0] == '#' || len(line) < 3 {
			/* skip comments, empty lines */
		} else if line[0] == '[' && line[len(line)-1:] == "]" {
			deviceName = line[1 : len(line)-1]
			config[deviceName] = map[string]string{}
		} else if strings.Contains(line, "=") {
			kv := strings.Split(line, "=")
			config[deviceName][strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])

		} else {
			log.Fatal("unknown config line: " + line)
		}

		if err != nil {
			break
		}
	}

	if _, ok := config["__global"]["port"]; !ok {
		config["__global"]["port"] = "80"
	}

	if _, ok := config["__global"]["mqtt_server"]; !ok {
		config["__global"]["mqtt_server"] = "127.0.0.1"
	}

	if _, ok := config["__global"]["webroot"]; !ok {
		config["__global"]["webroot"] = "/usr/share/dimmy"
	}

	return config["__global"], config, nil
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
