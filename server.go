package main

import (
    "bufio"
    "encoding/json"
    "errors"
    "fmt"
    "html/template"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

const cycleLength = 200

func main() {

    devices := make(map[string]DeviceInterface)

    config, deviceConfig, err := loadConfig()
    if err != nil {
        log.Fatal(err)
    }

    for key := range deviceConfig {
        if key != "__global" {
            switch deviceConfig[key]["type"] {
                case "sensor":
                    devices[key] = NewSensor(deviceConfig[key])
                case "zsensor":
                    devices[key] = NewZSensor(deviceConfig[key])
                case "switch":
                    devices[key] = NewSwitch(deviceConfig[key])
                case "light":
                    devices[key] = NewLight(deviceConfig[key])
                case "plug":
                    devices[key] = NewPlug(deviceConfig[key])
                case "thermostat":
                    devices[key] = NewThermostat(deviceConfig[key])
                default:
                    log.Println("Skipping device of unknown type '" + deviceConfig[key]["type"] + "'")
            }
        }
    }

    channel := make(chan SwitchRequest, 10)

    go eventLoop(devices, channel, config["mqtt_server"])

    assets := http.FileServer(http.Dir(config["webroot"] + "/assets"))
    http.Handle("/assets/", http.StripPrefix("/assets/", assets))
    http.Handle("/api/switch", http.HandlerFunc(ReceiveRequest(channel)))
    http.Handle("/api/status", http.HandlerFunc(ShowStatus(&devices)))
    http.Handle("/",  http.HandlerFunc(ShowDashboard(devices, channel, config["webroot"])))

    log.Fatal(http.ListenAndServe(":" + config["port"], nil))
}

func eventLoop(devices map[string]DeviceInterface, channel chan SwitchRequest, mqttServer string) {
    hostname, _ := os.Hostname()
    client := initMqtt(mqttServer, "goserver-" + hostname)

    for name, _ := range devices {
        client.Subscribe(devices[name].getMqttTopic(), 0, devices[name].getMessageHandler(channel, devices[name]))
    }

    for {
        time.Sleep(cycleLength * time.Millisecond)

        for ; len(channel) > 0 ; {
            request := <-channel
            if _, ok := devices[request.Device]; ok {

                devices[request.Device].processRequest(request);

            } else {
                log.Println("Unknown device [" + request.Device + "]")
            }
        }

        for name, _ := range devices {
            if _, ok := devices[name].UpdateValue(); ok {
                devices[name].PublishValue(client)
            }
        }

        for name, _ := range devices {
            if devices[name].getType() == "sensor" {
                request, ok := devices[name].getTimeoutRequest()
                if ok {
                    channel <- request
                }

            }
        }

    }
}

func ReceiveRequest(channel chan SwitchRequest) http.HandlerFunc {
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

        var request SwitchRequest

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

func ShowStatus(devices *map[string]DeviceInterface) http.HandlerFunc {
    return func(output http.ResponseWriter, request *http.Request) {
        jsonDevices, _ := json.Marshal(devices)
        fmt.Fprintf(output, string(jsonDevices))
    }
}

func ShowDashboard(devices map[string]DeviceInterface, channel chan SwitchRequest, webroot string) http.HandlerFunc {
    return func(output http.ResponseWriter, request *http.Request) {
        if request.Method == "POST" {
            err := request.ParseForm()
            if err == nil {
                var sr SwitchRequest
                sr.Duration = 0
                sr.Device = request.FormValue("device")
                target := request.FormValue("target")
                switch target {
                    case "on":
                        sr.Value = devices[sr.Device].getMax()
                    case "off":
                        sr.Value = devices[sr.Device].getMin()
                    case "+":
                        sr.Value = int(devices[sr.Device].getCurrent()) + 10
                    case "-":
                        sr.Value = int(devices[sr.Device].getCurrent()) - 10
                }
                channel <-sr
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

func loadConfig() (map[string]string, map[string]map[string]string, error) {
    config := map[string]map[string]string{}

    var filename string

    if _, err := os.Stat("/etc/dimmyd.conf"); err == nil {
        filename = "/etc/dimmyd.conf"
    } else if _, err := os.Stat("dimmyd.conf"); err == nil {
        filename = "dimmyd.conf"
    } else {
        return nil, nil, errors.New("Could not find config file /etc/dimmyd.conf")
    }

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

        if line[0] == '#' || len(line) < 3{
            /* skip comments, empty lines */
        } else if line[0] == '[' && line[len(line)-1:] == "]" {
            deviceName = line[1:len(line)-1]
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
