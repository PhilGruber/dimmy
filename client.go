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
    port := flag.Int("port", 8080, "port to connect to")

    value    := flag.Int("value", 100, "Value to set the device to [0-100]")
    device   := flag.String("device", "", "Device to send command to")
    duration := flag.Int("duration", 0, "Duration of the dimming curve (seconds)")

    flag.Parse()

    request := SwitchRequest{
        Device: *device,
        Command: "dim",
        Value: *value,
        Duration: *duration,
    }
    jsonRequest, _ := json.Marshal(request)
    url := fmt.Sprintf("http://%s:%d/api/switch", *host, *port)

    _, err := http.Post(url, "application/json", bytes.NewBuffer(jsonRequest))
    if err != nil {
        log.Fatal(err)
    }
}
