package core

import "log"

const debugEnabled = false

func D(s string) {
	if debugEnabled {
		log.Println(s)
	}
}
