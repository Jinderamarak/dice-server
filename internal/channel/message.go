package channel

import "encoding/json"

type Message struct {
	Variant string           `json:"variant"`
	Data    *json.RawMessage `json:"data"`
}

func CraftMessage(variant string, data interface{}) (*Message, error) {
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

func MustCraftMessage(variant string, data interface{}) *Message {
	message, err := CraftMessage(variant, data)
	if err != nil {
		panic(err)
	}
	return message
}

func (message *Message) Unmarshal(data []byte) error {
	return json.Unmarshal(data, message)
}

func (message *Message) Marshal() ([]byte, error) {
	return json.Marshal(message)
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
