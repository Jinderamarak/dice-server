package utility

import "io"

func CloseAndIgnore(c io.Closer) {
	_ = c.Close()
}
