package mocks

import "os"

var debug bool

func Init() {
	if os.Getenv("MOCKS_DEBUG") == "true" {
		debug = true
	} else {
		debug = false
	}
}
