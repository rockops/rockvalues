package main

import (
	"fmt"
	"os"
)

// Debug when helm is in debug mode
func Debug(msg string) {
	Fdebug("%s", msg)
}

func Fdebug(format string, args ...interface{}) {
	if os.Getenv("HELM_DEBUG") != "false" {
		fmt.Fprintf(os.Stderr, "get_values [debug] "+format+"\n", args...)
	}
}

func Ftrace(format string, args ...interface{}) {
	if os.Getenv("HELM_TRACE") != "" {
		fmt.Fprintf(os.Stderr, "get_values [trace] "+format+"\n", args...)
	}
}

func LogError(msg string) {
	Ferror(msg)
}

func Ferror(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\n** get_values ERROR ** : "+msg+"\n", args...)
}

func Warn(msg string) {
	Fwarn(msg)
}

func Fwarn(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\n** get_values WARN ** : "+msg+"\n", args...)
}
