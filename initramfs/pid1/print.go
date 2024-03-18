package pid1

import (
	"fmt"
	"sync"
)

var console sync.Mutex

func Printf(f string, a ...any) {
	console.Lock()
	defer console.Unlock()
	fmt.Printf(f, a...)
}

func MsgOK(f string, a ...any) {
	msg(tagOK, f, a...)
}

func MsgErr(f string, a ...any) {
	msg(tagError, f, a...)
}

func msg(tag string, f string, a ...any) {
	if tag == "" {
		tag = tagEmpty
	} else {
		tag = fmt.Sprintf("[%s]", tag)
	}
	Printf("%s %s\n", tag, fmt.Sprintf(f, a...))
}

const (
	asciiEsc   = "\u001B["
	asciiRed   = asciiEsc + "31;1m"
	asciiGreen = asciiEsc + "32;1m"
	asciiReset = asciiEsc + "0m"
	tagEmpty   = "      "
	tagError   = asciiRed + " FAIL " + asciiReset
	tagOK      = asciiGreen + "  OK  " + asciiReset
)
