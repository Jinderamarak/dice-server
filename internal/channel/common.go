package channel

import "errors"

const MessageLimit = 32

var ErrMessageLimit = errors.New("message limit exceeded")
var ErrClientClosed = errors.New("client is closed")
var ErrReadTimeout = errors.New("read timeout")
