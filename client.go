package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	core "github.com/PhilGruber/dimmy/core"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type listRequest struct {
	Value       float64 `json:"Value"`
	Type        string  `json:"Type"`
	Hidden      bool    `json:"Hidden"`
	Target      float64 `json:"Target"`
	LinkQuality *int    `json:"LinkQuality"`
	Battery     *int    `json:"Battery"`
}

func loadClientConfig() (*string, *int) {
	var filename string

	var port = 8080
	var host = "localhost"

	if _, err := os.Stat("/etc/dimmy.conf"); err == nil {
		filename = "/etc/dimmy.conf"
	} else if _, err := os.Stat("dimmyd.conf"); err == nil {
		filename = "dimmy.conf"
	} else {
		return &host, &port
	}

	file, err := os.Open(filename)

	if err != nil {
		return &host, &port
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println("Error: " + err.Error())
		}
	}(file)

	reader := bufio.NewReader(file)
	var line string
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)

		if strings.Contains(line, "=") {
			kv := strings.Split(line, "=")
			if kv[0] == "host" {
				host = kv[1]
			}
			if kv[0] == "port" {
				port, _ = strconv.Atoi(kv[1])
			}
		}
	}
	return &host, &port
}

func main() {

	host, port := loadClientConfig()
	host = flag.String("host", "", "hostname to connect to")
	port = flag.Int("port", *port, "port to connect to")

	value := flag.String("value", "100", "Value to send to device")
	device := flag.String("device", "", "Device to send command to")
	duration := flag.Int("duration", 0, "Duration of the dimming curve (seconds)")
	list := flag.Bool("list", false, "List devices and their status")

	flag.Parse()

	request := core.SwitchRequest{
		Device:   *device,
		Value:    *value,
		Duration: *duration,
	}

	jsonRequest, _ := json.Marshal(request)
	url := fmt.Sprintf("http://%s:%d/api/", *host, *port)

	if *list {
		fmt.Println("Getting device list from " + url)
		response, err := http.Get(url + "status")
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println("Error: " + err.Error())
			}
		}(response.Body)

		if err != nil {
			log.Println("Error: " + err.Error())
		}
		body, err := io.ReadAll(response.Body)

		var devices map[string]listRequest
		err = json.Unmarshal(body, &devices)
		if err != nil {
			log.Fatal("Error: " + err.Error())
		}
		for name, device := range devices {
			fmt.Printf("[%-12s] %-30s %5.1f", device.Type, name, device.Value)
			if device.LinkQuality != nil {
				fmt.Printf("\tSignal: %4d%%", *device.LinkQuality)
			} else {
				fmt.Printf("\t             ")
			}
			if device.Battery != nil {
				fmt.Printf("\tBattery: %4d%%", *device.Battery)
			}
			fmt.Println()
		}
	}

	_, err := http.Post(url+"switch", "application/json", bytes.NewBuffer(jsonRequest))
	if err != nil {
		log.Fatal("Error: " + err.Error())
	}
}
