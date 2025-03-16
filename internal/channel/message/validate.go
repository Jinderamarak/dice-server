package message

import "errors"

const maxDataLength = 1024

var (
	ErrNilMessage     = errors.New("nil message")
	ErrDataTooLong    = errors.New("data too long")
	ErrControlMessage = errors.New("control message")
)

func ValidateUserMessage(message *Message) error {
	if message == nil {
		return ErrNilMessage
	}

	if len(*message.Data) > maxDataLength {
		return ErrDataTooLong
	}

	if IsControlMessage(message) {
		return ErrControlMessage
	}

	return nil
}
