package devices

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

type recordingClient struct {
	mqtt.Client
	topics   []string
	messages []map[string]any
}

func (c *recordingClient) Publish(topic string, _ byte, _ bool, payload interface{}) mqtt.Token {
	var message map[string]any
	_ = json.Unmarshal(payload.([]byte), &message)
	c.topics = append(c.topics, topic)
	c.messages = append(c.messages, message)
	return completedToken{}
}

func TestZLightDecodedStateAndTelemetry(t *testing.T) {
	l := NewZLight(core.DeviceConfig{Transformer: "zigbee"})
	handler := l.GetMessageHandler(nil, l)
	handler(nil, &mockMessage{payload: []byte(`{"brightness":127,"state":"ON","battery":80,"linkquality":100}`)})
	require.Equal(t, l.ValueToPercentage(127), l.GetCurrent())
	require.Equal(t, 80, *l.Battery)
	require.Equal(t, 100, *l.LinkQuality)
	handler(nil, &mockMessage{payload: []byte(`{"brightness":127,"state":"OFF"}`)})
	require.Zero(t, l.GetCurrent())
	handler(nil, &mockMessage{payload: []byte(`{"brightness":"invalid"}`)})
	require.Zero(t, l.GetCurrent())
}

func TestZLightPublishesTransition(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		l := NewZLight(core.DeviceConfig{Topic: "zigbee/light", Transformer: "zigbee", Options: &core.ConfigOptions{Transition: &enabled}})
		l.ProcessRequest(core.SwitchRequest{Value: "50", Duration: 5})
		l.UpdateValue()
		client := &recordingClient{}
		l.PublishValue(client)
		require.Equal(t, []string{"zigbee/light/set"}, client.topics)
		if enabled {
			require.Equal(t, float64(5), client.messages[0]["transition"])
		} else {
			require.NotContains(t, client.messages[0], "transition")
		}
	}
}

func TestTransformerConfigRoundTrip(t *testing.T) {
	for _, original := range []DeviceInterface{
		NewLight(core.DeviceConfig{Transformer: "tasmota"}),
		NewZLight(core.DeviceConfig{Transformer: "zigbee"}),
		NewDevice(core.DeviceConfig{Transformer: "zigbee"}),
	} {
		config := original.GetConfig("saved")
		require.NotEmpty(t, config.Transformer)
		switch config.Type {
		case "light":
			require.Equal(t, "tasmota", config.Transformer)
		default:
			require.Equal(t, "zigbee", config.Transformer)
		}
	}
}

func TestDiscoveredLightTransformer(t *testing.T) {
	for _, tc := range []struct {
		topic       string
		data        map[string]any
		transformer string
	}{
		{"cmnd/lamp", map[string]any{"Dimmer": float64(50), "POWER": "ON"}, "tasmota"},
		{"zigbee/lamp", map[string]any{"brightness": float64(127), "state": "ON"}, "zigbee"},
	} {
		discovered := NewDeviceFromMessage(tc.topic, tc.data)
		config := discovered.GetConfig("saved")
		require.Equal(t, tc.transformer, config.Transformer)
		var saved DeviceInterface
		if tc.transformer == "tasmota" {
			saved = NewLight(config)
		} else {
			saved = NewZLight(config)
		}
		payload, err := json.Marshal(tc.data)
		require.NoError(t, err)
		saved.GetMessageHandler(nil, saved)(nil, &mockMessage{payload: payload})
		require.Positive(t, saved.GetCurrent())
	}
}

func TestDefaultTransformers(t *testing.T) {
	light := NewLight(core.DeviceConfig{})
	require.Equal(t, "tasmota", light.GetConfig("light").Transformer)
	light.GetMessageHandler(nil, light)(nil, &mockMessage{payload: []byte("50")})
	require.Equal(t, float64(50), light.GetCurrent())
	zlight := NewZLight(core.DeviceConfig{})
	require.Equal(t, "zigbee", zlight.GetConfig("zlight").Transformer)
	topic, _, err := zlight.transformer.PollMessage("zigbee/light", map[string]any{"brightness": 0})
	require.NoError(t, err)
	require.Equal(t, "zigbee/light/get", topic)
	generic := NewDevice(core.DeviceConfig{})
	require.Equal(t, "zigbee", generic.GetConfig("device").Transformer)
}

func TestUnknownTransformerFallsBackToZigbee(t *testing.T) {
	light := NewZLight(core.DeviceConfig{Transformer: "unknown"})
	topic, _, err := light.transformer.PollMessage("zigbee/light", nil)
	require.NoError(t, err)
	require.Equal(t, "zigbee/light/get", topic)
	light.GetMessageHandler(nil, light)(nil, &mockMessage{payload: []byte(`{"brightness":127}`)})
	require.Equal(t, light.ValueToPercentage(127), light.GetCurrent())
}

func TestCbusLightRamps(t *testing.T) {
	for _, kind := range []string{"light", "zlight"} {
		t.Run(kind, func(t *testing.T) {
			config := core.DeviceConfig{Topic: "cbus/lighting/1", Transformer: "cbus"}
			var light DeviceInterface
			if kind == "light" {
				light = NewLight(config)
			} else {
				light = NewZLight(config)
			}
			require.Equal(t, 255, light.GetMax())
			require.Equal(t, config.Topic, light.GetMqttStateTopic())
			for _, tc := range []struct {
				percentage string
				level      float64
			}{{"100", 255}, {"0", 0}} {
				light.ProcessRequest(core.SwitchRequest{Value: tc.percentage, Duration: 4})
				light.UpdateValue()
				client := &recordingClient{}
				light.PublishValue(client)
				require.Equal(t, []string{config.Topic + "/set"}, client.topics)
				require.Equal(t, []map[string]any{{"command": "ramp", "value": tc.level, "time": float64(4)}}, client.messages)
			}
			require.NotPanics(t, func() { light.PollValue(nil) })
		})
	}
}

type completedToken struct{}

func (completedToken) Wait() bool                     { return true }
func (completedToken) WaitTimeout(time.Duration) bool { return true }
func (completedToken) Done() <-chan struct{}          { done := make(chan struct{}); close(done); return done }
func (completedToken) Error() error                   { return nil }

func TestCbusGenericDeviceCommands(t *testing.T) {
	for _, tc := range []struct {
		key, value string
		kind       core.ControlType
		want       map[string]any
	}{
		{"state", "ON", core.ControlTypeBool, map[string]any{"command": "on"}},
		{"state", "OFF", core.ControlTypeBool, map[string]any{"command": "off"}},
		{"brightness", "255", core.ControlTypeDimmable, map[string]any{"command": "ramp", "value": float64(255), "time": float64(4)}},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			controls := []core.Control{{Name: tc.key, Type: tc.kind}}
			device := NewDevice(core.DeviceConfig{Topic: "cbus/lighting/1", Transformer: "cbus", Options: &core.ConfigOptions{Controls: &controls}})
			device.ProcessRequest(core.SwitchRequest{Key: tc.key, Value: tc.value, Duration: 4})
			client := &recordingClient{}
			device.PublishValue(client)
			require.Equal(t, []string{"cbus/lighting/1/set"}, client.topics)
			require.Equal(t, []map[string]any{tc.want}, client.messages)
			require.False(t, device.Controls[0].NeedsSending)
			require.NotPanics(t, func() { device.PollValue(nil) })
		})
	}
}

func TestCbusLightStateUpdates(t *testing.T) {
	for _, kind := range []string{"light", "zlight"} {
		t.Run(kind, func(t *testing.T) {
			var light DeviceInterface
			config := core.DeviceConfig{Transformer: "cbus"}
			if kind == "light" {
				light = NewLight(config)
			} else {
				light = NewZLight(config)
			}
			handler := light.GetMessageHandler(nil, light)
			for _, tc := range []struct {
				payload    string
				percentage float64
			}{
				{`{"command":"on"}`, 100},
				{`{"command":"ramp","value":128,"time":4}`, 51},
				{`{"command":"off"}`, 0},
			} {
				handler(nil, &mockMessage{payload: []byte(tc.payload)})
				require.InDelta(t, tc.percentage, light.GetCurrent(), 1)
			}
		})
	}
}
