package main

import (
    "encoding/json"
    "net/http"
    "fmt"
    "bytes"
    "flag"
    "log"
)

func main() {
    host := flag.String("host", "127.0.0.1", "hostname to connect to")
    port := flag.Int("post", 8080, "port to connect to")

    value  := flag.Int("value", 100, "Value to set the device to [0-100]")
    device := flag.String("device", "", "Device to send command to")
    delay  := flag.Int("delay", 0, "Delay between steps")

    flag.Parse()

    request := SwitchRequest{
        Device: *device,
        Command: "dim",
        Value: *value,
        Delay: *delay,
    }
    jsonRequest, _ := json.Marshal(request)
    url := fmt.Sprintf("http://%s:%d/switch", *host, *port)

    log.Println(string(jsonRequest))
    log.Println(url)
    result, err := http.Post(url, "application/json", bytes.NewBuffer(jsonRequest))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}

type SwitchRequest struct {
    Device string `json:"device"`
    Command string `json:"command"`
    Value int `json:"value"`
    Delay int `json:"delay"`
}
