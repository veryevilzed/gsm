package gsm

import (
	"fmt"
	"strings"
)

// DebugMode включает/выключает режим отладки
var DebugMode = false

// EnableDebug включает режим отладки
func EnableDebug() {
	DebugMode = true
}

// DisableDebug выключает режим отладки
func DisableDebug() {
	DebugMode = false
}

// debugLog выводит отладочную информацию
func debugLog(format string, args ...interface{}) {
	if DebugMode {
		fmt.Printf("[GSM DEBUG] "+format+"\n", args...)
	}
}

// debugResponse выводит ответ модема в читаемом виде
func debugResponse(command, response string) {
	if DebugMode {
		fmt.Printf("[GSM DEBUG] Command: %s\n", command)
		fmt.Printf("[GSM DEBUG] Response:\n")

		// Показываем специальные символы
		readable := strings.ReplaceAll(response, "\r", "\\r")
		readable = strings.ReplaceAll(readable, "\n", "\\n\n")

		lines := strings.Split(readable, "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("  %s\n", line)
			}
		}
		fmt.Println()
	}
}

// FormatResponse форматирует ответ для отображения
func FormatResponse(response string) string {
	var result strings.Builder

	result.WriteString("Raw response:\n")

	// Показываем байты
	result.WriteString("Bytes: [")
	for i, b := range []byte(response) {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(fmt.Sprintf("%02X", b))
	}
	result.WriteString("]\n")

	// Показываем читаемый вид
	result.WriteString("Readable:\n")
	readable := strings.ReplaceAll(response, "\r", "\\r")
	readable = strings.ReplaceAll(readable, "\n", "\\n")
	result.WriteString(readable)
	result.WriteString("\n")

	// Показываем построчно
	result.WriteString("Lines:\n")
	lines := strings.Split(response, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result.WriteString(fmt.Sprintf("  [%d]: '%s'\n", i, line))
		}
	}

	return result.String()
}
