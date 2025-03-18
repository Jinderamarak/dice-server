package utility

import (
	"io"
	"log"
)

func CloseAndIgnore(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Println("Error closing resource:", err)
	}
}
