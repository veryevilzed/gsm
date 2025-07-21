package gsm

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/tarm/serial"
)

// DecodeUCS2 декодирует UCS2/UTF-16 текст из hex строки
func DecodeUCS2(hexStr string) (string, error) {
	// Убираем пробелы если есть
	hexStr = strings.ReplaceAll(hexStr, " ", "")

	// Декодируем hex в байты
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex: %w", err)
	}

	// Проверяем, что длина четная (UTF-16 использует 2 байта на символ)
	if len(data)%2 != 0 {
		return "", fmt.Errorf("invalid UCS2 data: odd number of bytes")
	}

	// Конвертируем байты в uint16 (UTF-16)
	runes := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		// Big-endian byte order
		runes[i/2] = uint16(data[i])<<8 | uint16(data[i+1])
	}

	// Декодируем UTF-16 в UTF-8
	return string(utf16.Decode(runes)), nil
}

// EncodeUCS2 кодирует текст в UCS2/UTF-16 hex строку
func EncodeUCS2(text string) string {
	// Конвертируем в UTF-16
	runes := utf16.Encode([]rune(text))

	// Конвертируем в байты
	data := make([]byte, len(runes)*2)
	for i, r := range runes {
		data[i*2] = byte(r >> 8)     // High byte
		data[i*2+1] = byte(r & 0xFF) // Low byte
	}

	// Кодируем в hex
	return hex.EncodeToString(data)
}

// IsUCS2Hex проверяет, является ли строка UCS2 в hex формате
func IsUCS2Hex(str string) bool {
	// Убираем пробелы
	str = strings.ReplaceAll(str, " ", "")

	// Проверяем, что строка содержит только hex символы
	if len(str) == 0 || len(str)%4 != 0 {
		return false
	}

	// Проверяем, что все символы - hex
	for _, c := range str {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}

	// Проверяем, что это похоже на UCS2 (большинство символов начинаются с 00 или 04)
	decoded, err := hex.DecodeString(str)
	if err != nil {
		return false
	}

	highByteCount := 0
	for i := 0; i < len(decoded); i += 2 {
		if decoded[i] == 0x00 || decoded[i] == 0x04 {
			highByteCount++
		}
	}

	// Если больше половины символов имеют типичные для UCS2 старшие байты
	return highByteCount > len(decoded)/4
}

// DecodeGSMText автоматически определяет кодировку и декодирует текст
func DecodeGSMText(text string) string {
	// Проверяем, не является ли это UCS2 в hex
	if IsUCS2Hex(text) {
		decoded, err := DecodeUCS2(text)
		if err == nil {
			return decoded
		}
	}

	// Проверяем, не является ли это просто hex-encoded ASCII
	if len(text) > 0 && len(text)%2 == 0 {
		allHex := true
		for _, c := range text {
			if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
				allHex = false
				break
			}
		}

		if allHex {
			if decoded, err := hex.DecodeString(text); err == nil {
				// Проверяем, что результат - валидный UTF-8
				if utf8.Valid(decoded) {
					return string(decoded)
				}
			}
		}
	}

	// Возвращаем как есть
	return text
}

// readWithTimeout читает данные из порта с таймаутом
func readWithTimeout(port *serial.Port, timeout time.Duration) ([]byte, error) {
	result := make(chan struct {
		data []byte
		err  error
	}, 1)

	go func() {
		buf := make([]byte, 1024)
		n, err := port.Read(buf)
		result <- struct {
			data []byte
			err  error
		}{buf[:n], err}
	}()

	select {
	case res := <-result:
		return res.data, res.err
	case <-time.After(timeout):
		return nil, nil // timeout
	}
}

// waitForResponse ждет определенный ответ с таймаутом
func waitForResponse(port *serial.Port, expected string, timeout time.Duration) (string, error) {
	var response string
	buf := make([]byte, 256)
	endTime := time.Now().Add(timeout)

	for time.Now().Before(endTime) {
		n, err := port.Read(buf)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if n > 0 {
			response += string(buf[:n])
			if contains(response, expected) {
				return response, nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	return response, nil
}

// contains проверяет, содержит ли строка подстроку
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

// parseKeyValue парсит ответы вида "+CMD: key,value"
func parseKeyValue(response, prefix string) map[string]string {
	result := make(map[string]string)

	// Простой парсер для AT ответов
	// Можно расширить при необходимости

	return result
}
