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
    "math"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

const cycleLength = 200

func main() {

    devices := make(map[string]*Device)
    sensors := make(map[string]*TuyaSensor)

    deviceConfig, err := loadConfig("devices.conf")
    if err != nil {
        log.Fatal(err)
    }

    for key := range deviceConfig {
        if deviceConfig[key]["type"] == "sensor" {
            sensors[key] = NewTuyaSensor(deviceConfig[key])
        } else {
            devices[key] = NewDevice(deviceConfig[key])
        }
    }

    channel := make(chan SwitchRequest, 10)

    go eventLoop(devices, sensors, channel, "192.168.178.48")

    assets := http.FileServer(http.Dir("assets"))
    http.Handle("/assets/", http.StripPrefix("/assets/", assets))
    http.Handle("/api/switch", http.HandlerFunc(ReceiveRequest(channel)))
    http.Handle("/api/status", http.HandlerFunc(ShowStatus(&devices)))
    http.Handle("/",  http.HandlerFunc(ShowDashboard(devices, channel)))

    log.Fatal(http.ListenAndServe(":8080", nil))
}

func eventLoop(devices map[string]*Device, sensors map[string]*TuyaSensor, channel chan SwitchRequest, mqttServer string) {
    hostname, _ := os.Hostname()
    mqtt := initMqtt(mqttServer, "goserver-" + hostname)

    for name, _ := range sensors {
        mqtt.Subscribe(sensors[name].MqttTopic, 0, TuyaSensorMessageHandler(channel, sensors[name]))
    }

    for {
        time.Sleep(cycleLength * time.Millisecond)

        for ; len(channel) > 0 ; {
            request := <-channel
//            log.Println(request)
            if _, ok := devices[request.Device]; ok {
                request.Value = int(math.Min(float64(request.Value), float64(devices[request.Device].Max)));
                request.Value = int(math.Max(float64(request.Value), float64(devices[request.Device].Min)));
                log.Println("Dimming " + request.Device + " to " + strconv.Itoa(request.Value))

                devices[request.Device].Target = request.Value
                diff := int(math.Abs(devices[request.Device].Current - float64(request.Value)))
                var step float64
                cycles := request.Duration * 1000/cycleLength
                if request.Duration == 0 {
                    step = float64(diff)
                } else {
                    step = float64(diff) / float64(cycles)
                }

//                log.Printf("steps per cycle = %f (%d steps / %d cycles)", step, diff, cycles)

                log.Printf("Dimming %d steps in %d seconds = %f steps per cycle", diff, request.Duration, step)
                devices[request.Device].Step  = step
            } else if _, ok := sensors[request.Device]; ok {
                sensors[request.Device].Value = request.Value
            } else {
                log.Println("Unknown device [" + request.Device + "]")
            }
        }

        for name, _ := range devices {
            if value, ok := devices[name].UpdateValue(); ok {
                devices[name].Current = value
                tt := time.Now()
                if int(math.Round(value)) != devices[name].LastSent {
                    devices[name].LastChanged = &tt
                    devices[name].LastSent = int(math.Round(value))
//                    log.Printf("Setting %s to %f", int(math.Round(value)))
                    mqtt.Publish(devices[name].MqttTopic, 0, false, strconv.Itoa(int(math.Round(value))))
                }
            }
        }

        for name, _ := range sensors {
            if sensors[name].Active {
                if (sensors[name].LastChanged.Local().Add(time.Second * time.Duration(sensors[name].Timeout))).Before(time.Now()) {
                    log.Println("Timeout")
                    sensors[name].Active = false

                    var request SwitchRequest
                    request.Device   = sensors[name].Target
                    request.Value    = sensors[name].TargetOff
                    request.Duration = sensors[name].TargetOffDuration
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
                sr.Duration = 0
                sr.Device = request.FormValue("device")
                target := request.FormValue("target")
                switch target {
                    case "on":
                        sr.Value = devices[sr.Device].Max
                    case "off":
                        sr.Value = devices[sr.Device].Min
                    case "+":
                        sr.Value = int(devices[sr.Device].Current) + 10
                    case "-":
                        sr.Value = int(devices[sr.Device].Current) - 10
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
