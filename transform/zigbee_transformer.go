package transform

import "encoding/json"

type ZigbeeTransformer struct {
	Transformer
}

func (t ZigbeeTransformer) Encode(topic string, data map[string]any) (string, []byte, error) {
	message, err := json.Marshal(data)
	return topic + "/set", message, err
}

func (t ZigbeeTransformer) Decode(data []byte) (map[string]any, error) {
	var result map[string]any
	err := json.Unmarshal(data, &result)
	return result, err
}

func (t ZigbeeTransformer) PollMessage(topic string, data map[string]any) (string, []byte, error) {
	msg, err := json.Marshal(data)
	return topic + "/get", msg, err
}
