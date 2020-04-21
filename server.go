package main

import (
    "log"
    "net/http"
    "fmt"
    "encoding/json"
    "io/ioutil"
    "time"
    "bufio"
    "os"
    "strings"
    "html/template"
    "math"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

const cycleLength = 200

func main() {

    devices := make(map[string]DeviceInterface)

    deviceConfig, err := loadConfig("devices.conf")
    if err != nil {
        log.Fatal(err)
    }

    for key := range deviceConfig {
        switch deviceConfig[key]["type"] {
            case "sensor":
                devices[key] = NewSensor(deviceConfig[key])
            case "light":
                devices[key] = NewLight(deviceConfig[key])
            default:
                log.Println("Skipping device of unknown type '" + deviceConfig[key]["type"] + "'")
        }
    }

    channel := make(chan SwitchRequest, 10)

    go eventLoop(devices, channel, "192.168.178.48")

    assets := http.FileServer(http.Dir("assets"))
    http.Handle("/assets/", http.StripPrefix("/assets/", assets))
    http.Handle("/api/switch", http.HandlerFunc(ReceiveRequest(channel)))
    http.Handle("/api/status", http.HandlerFunc(ShowStatus(&devices)))
    http.Handle("/",  http.HandlerFunc(ShowDashboard(devices, channel)))

    log.Fatal(http.ListenAndServe(":8080", nil))
}

func eventLoop(devices map[string]DeviceInterface, channel chan SwitchRequest, mqttServer string) {
    hostname, _ := os.Hostname()
    mqtt := initMqtt(mqttServer, "goserver-" + hostname)

    for name, _ := range devices {
        if devices[name].getType() == "sensor" {
            mqtt.Subscribe(devices[name].getMqttTopic(), 0, SensorMessageHandler(channel, devices[name]))
        }
    }

    for {
        time.Sleep(cycleLength * time.Millisecond)

        for ; len(channel) > 0 ; {
            request := <-channel
            if _, ok := devices[request.Device]; ok {
                request.Value = int(math.Min(float64(request.Value), float64(devices[request.Device].getMax())));
                request.Value = int(math.Max(float64(request.Value), float64(devices[request.Device].getMin())));

                devices[request.Device].setTarget(request.Value)
                diff := int(math.Abs(devices[request.Device].getCurrent() - float64(request.Value)))
                var step float64
                cycles := request.Duration * 1000/cycleLength
                if request.Duration == 0 {
                    step = float64(diff)
                } else {
                    step = float64(diff) / float64(cycles)
                }

                log.Printf("Dimming %d steps in %d seconds (%f steps per cycle)", diff, request.Duration, step)
                devices[request.Device].setStep(step)

            } else {
                log.Println("Unknown device [" + request.Device + "]")
            }
        }

        for name, _ := range devices {
            if ok := devices[name].UpdateValue(); ok {
                devices[name].PublishValue(mqtt)
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

func ShowDashboard(devices map[string]DeviceInterface, channel chan SwitchRequest) http.HandlerFunc {
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
                devices[sr.Device].setTarget(sr.Value)
                channel <-sr
            }
        }

        templ, _ := template.ParseFiles("html/dashboard.html")
        templ.Execute(output, devices)
    }
}

func loadConfig(filename string) (map[string]map[string]string, error) {
    config := map[string]map[string]string{}

    file, err := os.Open(filename)
    defer file.Close()

    if err != nil {
        return nil,err
    }

    reader := bufio.NewReader(file)

    var line string
    var deviceName string
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

    return config, nil
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
