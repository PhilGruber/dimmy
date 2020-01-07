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
    "strconv"
    "html/template"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {

    devices := make(map[string]*Device)

    deviceConfig, err := loadConfig("devices.conf")
    if err != nil {
        log.Fatal(err)
    }

    for key := range deviceConfig {
        devices[key] = NewDevice(deviceConfig[key])
    }

    channel := make(chan SwitchRequest, 10)

    go eventLoop(devices, channel, "192.168.178.48")

    http.Handle("/api/switch", http.HandlerFunc(ReceiveRequest(channel)))
    http.Handle("/api/status", http.HandlerFunc(ShowStatus(&devices)))
    http.Handle("/",  http.HandlerFunc(ShowDashboard(devices, channel)))
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func eventLoop(devices map[string]*Device, channel chan SwitchRequest, mqttServer string) {
    mqtt := initMqtt(mqttServer, "goserver")
    for {
        time.Sleep(100 * time.Millisecond)

        for ; len(channel) > 0 ; {
            request := <-channel
            log.Println(request)
            if _, ok := devices[request.Device]; ok {
                log.Println("Dimming " + request.Device + " to " + strconv.Itoa(request.Value))
                devices[request.Device].Target = request.Value
                devices[request.Device].Delay = time.Duration(request.Delay) * time.Second
            }
        }

        for name, _ := range devices {
            if value, ok := devices[name].UpdateValue(); ok {
                log.Println("Setting " + name + " to " + strconv.Itoa(value))
                devices[name].Current = value
                tt := time.Now()
                devices[name].LastChanged = &tt
                mqtt.Publish(devices[name].MqttTopic, 0, false, strconv.Itoa(value))
            }
        }
    }
}

func ReceiveRequest(channel chan SwitchRequest) http.HandlerFunc {
        return func(output http.ResponseWriter, httpRequest *http.Request) {

        jsonResponse:= func(result bool, request interface{}, message string) string {
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

        s := string(body[:])
        log.Println("Received " + s)

        var request SwitchRequest

        err = json.Unmarshal(body, &request)

        if err != nil {
            log.Println("Error: ", err)
            fmt.Fprintf(output, "Invalid JSON data")
            return
        }
        channel <- request
        fmt.Fprintf(output, jsonResponse(true, request, fmt.Sprintf("Sent %s command to %s", request.Command, request.Device)))
    }
}

func ShowStatus(devices *map[string]*Device) http.HandlerFunc {
    return func(output http.ResponseWriter, request *http.Request) {
        jsonDevices, _ := json.Marshal(devices)
        fmt.Fprintf(output, string(jsonDevices))
    }
}

func ShowDashboard(devices map[string]*Device, channel chan SwitchRequest) http.HandlerFunc {
    return func(output http.ResponseWriter, request *http.Request) {
        if request.Method == "POST" {
            err := request.ParseForm()
            if err == nil {
                var sr SwitchRequest
                sr.Delay = 0
                sr.Device = request.FormValue("device")
                target := request.FormValue("target")
                switch target {
                    case "on":
                        sr.Value = 100
                    case "off":
                        sr.Value = 0
                    case "+":
                        sr.Value = devices[sr.Device].Target + 10
                    case "-":
                        sr.Value = devices[sr.Device].Target - 10
                }
                devices[sr.Device].Target = sr.Value
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

type SwitchRequest struct {
    Device string
    Command string
    Value int
    Delay int
}
