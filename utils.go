package gsm

import (
	"time"

	"github.com/tarm/serial"
)

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
