package client

import "encoding/json"

type Message struct {
	Variant string           `json:"variant"`
	Data    *json.RawMessage `json:"data"`
}

func MarshalMessage(variant string, data interface{}) (*Message, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	message := Message{
		Variant: variant,
		Data:    (*json.RawMessage)(&raw),
	}
	return &message, nil
}

func MustMarshalMessage(variant string, data interface{}) *Message {
	message, err := MarshalMessage(variant, data)
	if err != nil {
		panic(err)
	}
	return message
}

func (message *Message) UnmarshalData(data interface{}) error {
	return json.Unmarshal(*message.Data, data)
}

func (message *Message) MustUnmarshalData(data interface{}) {
	err := message.UnmarshalData(data)
	if err != nil {
		panic(err)
	}
}
