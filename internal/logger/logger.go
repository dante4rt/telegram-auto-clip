package logger

import (
	"fmt"
	"log"
	"os"
)

var l = log.New(os.Stderr, "", log.LstdFlags)

func Info(msg string, args ...any) {
	if len(args) > 0 {
		l.Printf("[INFO] "+msg, args...)
	} else {
		l.Println("[INFO] " + msg)
	}
	os.Stderr.Sync()
}

func Error(msg string, args ...any) {
	if len(args) > 0 {
		l.Printf("[ERROR] "+msg, args...)
	} else {
		l.Println("[ERROR] " + msg)
	}
	os.Stderr.Sync()
}

func Debug(msg string, args ...any) {
	if len(args) > 0 {
		l.Printf("[DEBUG] "+msg, args...)
	} else {
		l.Println("[DEBUG] " + msg)
	}
	os.Stderr.Sync()
}

// Print is a simple wrapper for immediate output
func Print(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Stderr.Sync()
}
