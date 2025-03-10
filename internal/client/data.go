package client

import "encoding/json"

type Message struct {
	Variant int
	data    *json.RawMessage
}

func (message *Message) Unmarshal(data interface{}) error {
	return json.Unmarshal(*message.data, data)
}

func Marshal(variant int, data interface{}) (Message, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Message{}, err
	}

	message := Message{
		Variant: variant,
		data:    (*json.RawMessage)(&raw),
	}
	return message, nil
}
