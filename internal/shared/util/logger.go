package util

import (
	"fmt"
	"log"
	"os"
	"time"
)

// --- COLORS ---
var (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

// --- LOGGER STRUCT ---
type Logger struct {
	std *log.Logger
}

func New() *Logger {
	return &Logger{
		std: log.New(os.Stdout, "", 0), // we’ll print our own timestamp
	}
}

// --- LOG HELPERS ---

func (l *Logger) Info(instance, message string) {
	l.printf(Green, "INFO", instance, message)
}

func (l *Logger) Warn(instance, message string) {
	l.printf(Yellow, "WARN", instance, message)
}

func (l *Logger) Error(instance string, err error) {
	l.printf(Red, "ERROR", instance, err.Error())
}

func (l *Logger) Fatal(instance string, err error) {
	l.printf(Red, "FATAL", instance, err.Error())
	os.Exit(1)
}

func (l *Logger) OK(instance, message string) {
	l.printf(Green, "OK", instance, message)
}

func (l *Logger) printf(color, level, instance, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	l.std.Printf("%s|%s|%s %s%-5s%s | %-15s | %s\n",
		Reset, timestamp, color, level, Reset, "", instance, message)
}

// --- HTTP LOGGING ---

func (l *Logger) HTTP(status int, elapsed time.Duration, host, method, path string) {
	coloredStatus := paintStatus(status)
	coloredMethod := paintMethod(method)
	l.std.Printf("|%s| %7s | %-20s | %s %s\n",
		coloredStatus, elapsed, host, coloredMethod, path)
}

// --- COLOR HELPERS ---

func paintMethod(method string) string {
	switch method {
	case "GET":
		return Blue + fmt.Sprintf("%-6s", method) + Reset
	case "POST":
		return Green + fmt.Sprintf("%-6s", method) + Reset
	case "PUT":
		return Magenta + fmt.Sprintf("%-6s", method) + Reset
	case "DELETE":
		return Red + fmt.Sprintf("%-6s", method) + Reset
	case "OPTIONS":
		return Yellow + fmt.Sprintf("%-6s", method) + Reset
	default:
		return White + fmt.Sprintf("%-6s", method) + Reset
	}
}

func paintStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return Green + fmt.Sprintf("%d", code) + Reset
	case code >= 300 && code < 400:
		return Cyan + fmt.Sprintf("%d", code) + Reset
	case code >= 400 && code < 500:
		return Yellow + fmt.Sprintf("%d", code) + Reset
	case code >= 500:
		return Red + fmt.Sprintf("%d", code) + Reset
	default:
		return White + fmt.Sprintf("%d", code) + Reset
	}
}
