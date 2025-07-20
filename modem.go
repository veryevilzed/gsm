package gsm

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
)

// Modem представляет GSM модем
type Modem struct {
	port          *serial.Port
	config        *serial.Config
	mu            sync.Mutex
	eventChan     chan Event
	stopEventsCh  chan struct{}
	eventsEnabled bool
}

// ModemInfo содержит информацию о модеме
type ModemInfo struct {
	Port         string
	Manufacturer string
	Model        string
	Revision     string
	IMEI         string
	Description  string
}

// GetAvailableModems возвращает список доступных модемов
func GetAvailableModems() ([]ModemInfo, error) {
	var modems []ModemInfo

	switch runtime.GOOS {
	case "linux":
		// Ищем устройства в /dev/ttyUSB* и /dev/ttyACM*
		patterns := []string{
			"/dev/ttyUSB",
			"/dev/ttyACM",
		}
		for _, pattern := range patterns {
			for i := 0; i < 10; i++ {
				port := fmt.Sprintf("%s%d", pattern, i)
				if info := tryOpenModem(port); info != nil {
					modems = append(modems, *info)
				}
			}
		}

	case "darwin": // macOS
		// Ищем устройства в /dev/tty.usbserial* и /dev/tty.usbmodem*
		patterns := []string{
			"/dev/tty.usbserial",
			"/dev/tty.usbmodem",
			"/dev/cu.usbserial",
			"/dev/cu.usbmodem",
		}
		for _, pattern := range patterns {
			for i := 0; i < 10; i++ {
				port := fmt.Sprintf("%s%d", pattern, i)
				if info := tryOpenModem(port); info != nil {
					modems = append(modems, *info)
				}
			}
		}

	case "windows":
		// Ищем COM порты
		for i := 1; i <= 20; i++ {
			port := fmt.Sprintf("COM%d", i)
			if info := tryOpenModem(port); info != nil {
				modems = append(modems, *info)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return modems, nil
}

// tryOpenModem пытается открыть модем и получить его информацию
func tryOpenModem(port string) *ModemInfo {
	config := &serial.Config{
		Name:        port,
		Baud:        115200,
		ReadTimeout: time.Second,
	}

	serialPort, err := serial.OpenPort(config)
	if err != nil {
		return nil
	}
	defer serialPort.Close()

	// Проверяем, отвечает ли устройство на AT команды
	serialPort.Flush()
	_, err = serialPort.Write([]byte("AT\r\n"))
	if err != nil {
		return nil
	}

	// Читаем ответ
	buf := make([]byte, 128)
	n, err := serialPort.Read(buf)
	if err != nil || n == 0 || !strings.Contains(string(buf[:n]), "OK") {
		return nil
	}

	info := &ModemInfo{
		Port: port,
	}

	// Вспомогательная функция для отправки команд
	sendCmd := func(cmd string) string {
		serialPort.Flush()
		serialPort.Write([]byte(cmd + "\r\n"))
		time.Sleep(100 * time.Millisecond)

		buf := make([]byte, 256)
		n, _ := serialPort.Read(buf)
		if n > 0 {
			return string(buf[:n])
		}
		return ""
	}

	// Получаем информацию о производителе
	if resp := sendCmd("AT+CGMI"); resp != "" {
		info.Manufacturer = extractResponse(resp)
	}

	// Получаем модель
	if resp := sendCmd("AT+CGMM"); resp != "" {
		info.Model = extractResponse(resp)
	}

	// Получаем IMEI
	if resp := sendCmd("AT+CGSN"); resp != "" {
		info.IMEI = extractResponse(resp)
	}

	info.Description = fmt.Sprintf("%s %s", info.Manufacturer, info.Model)

	return info
}

// New создает новый экземпляр модема
func New(port string, baudRate int) (*Modem, error) {
	config := &serial.Config{
		Name:        port,
		Baud:        baudRate,
		ReadTimeout: time.Second * 5,
	}

	serialPort, err := serial.OpenPort(config)
	if err != nil {
		return nil, fmt.Errorf("failed to open port: %w", err)
	}

	m := &Modem{
		port:          serialPort,
		config:        config,
		eventChan:     make(chan Event, 100),
		stopEventsCh:  make(chan struct{}),
		eventsEnabled: false,
	}

	// Инициализация модема
	if err := m.initialize(); err != nil {
		serialPort.Close()
		return nil, fmt.Errorf("failed to initialize modem: %w", err)
	}

	return m, nil
}

// initialize выполняет базовую инициализацию модема
func (m *Modem) initialize() error {
	// Сброс до заводских настроек
	if _, err := m.SendCommand("ATZ", time.Second*2); err != nil {
		return err
	}

	// Отключаем эхо
	if _, err := m.SendCommand("ATE0", time.Second); err != nil {
		return err
	}

	// Устанавливаем текстовый режим для SMS
	if _, err := m.SendCommand("AT+CMGF=1", time.Second); err != nil {
		return err
	}

	// Включаем отчеты об ошибках
	if _, err := m.SendCommand("AT+CMEE=1", time.Second); err != nil {
		return err
	}

	return nil
}

// Close закрывает соединение с модемом
func (m *Modem) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Останавливаем события если они запущены
	if m.eventsEnabled {
		close(m.stopEventsCh)
		m.eventsEnabled = false
	}

	if m.port != nil {
		return m.port.Close()
	}

	return nil
}

// SendCommand отправляет AT команду и ждет ответ
func (m *Modem) SendCommand(cmd string, timeout time.Duration) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.sendCommand(cmd, timeout)
}

// sendCommand внутренний метод для отправки команд (без блокировки)
func (m *Modem) sendCommand(cmd string, timeout time.Duration) (string, error) {
	// Очищаем буфер перед отправкой
	m.port.Flush()

	// Отправляем команду
	_, err := m.port.Write([]byte(cmd + "\r\n"))
	if err != nil {
		return "", fmt.Errorf("failed to write command: %w", err)
	}

	// Читаем ответ
	return m.readResponse(timeout)
}

// readResponse читает ответ от модема
func (m *Modem) readResponse(timeout time.Duration) (string, error) {
	var response strings.Builder
	buf := make([]byte, 1024)
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		// Читаем доступные данные
		n, err := m.port.Read(buf)
		if err != nil {
			// Если уже что-то прочитали и есть финальный ответ, возвращаем
			if response.Len() > 0 {
				responseStr := response.String()
				if strings.Contains(responseStr, "OK") ||
					strings.Contains(responseStr, "ERROR") {
					return responseStr, nil
				}
			}
			// Небольшая пауза перед следующей попыткой
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if n > 0 {
			response.Write(buf[:n])
			responseStr := response.String()

			// Проверяем на финальные ответы
			if strings.Contains(responseStr, "\r\nOK\r\n") ||
				strings.Contains(responseStr, "\r\nERROR\r\n") ||
				strings.Contains(responseStr, "\r\n+CME ERROR") ||
				strings.Contains(responseStr, "\r\n+CMS ERROR") {
				return responseStr, nil
			}
		}

		// Небольшая пауза между чтениями
		time.Sleep(10 * time.Millisecond)
	}

	if response.Len() == 0 {
		return "", errors.New("timeout waiting for response")
	}
	return response.String(), nil
}

// extractResponse извлекает чистый ответ из AT команды
func extractResponse(response string) string {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "OK") &&
			!strings.HasPrefix(line, "ERROR") &&
			!strings.HasPrefix(line, "AT") {
			// Удаляем префикс команды если есть
			if idx := strings.Index(line, ":"); idx != -1 {
				return strings.TrimSpace(line[idx+1:])
			}
			return line
		}
	}
	return ""
}

// GetEventChannel возвращает канал событий
func (m *Modem) GetEventChannel() (<-chan Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.eventsEnabled {
		return nil, fmt.Errorf("event listener is not running, call StartEventListener() first")
	}

	return m.eventChan, nil
}
