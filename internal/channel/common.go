package channel

import "errors"

const MessageLimit = 32

var (
	ErrMessageLimit = errors.New("message limit exceeded")
	ErrClientClosed = errors.New("client is closed")
	ErrReadTimeout  = errors.New("read timeout")
)
