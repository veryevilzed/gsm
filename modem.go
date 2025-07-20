package gsm

import (
	"bufio"
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
	port         *serial.Port
	config       *serial.Config
	mu           sync.Mutex
	eventChan    chan Event
	stopEventsCh chan struct{}
	reader       *bufio.Reader
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

	// Создаем временный модем для проверки
	m := &Modem{
		port:   serialPort,
		config: config,
		reader: bufio.NewReader(serialPort),
	}

	// Проверяем, отвечает ли устройство на AT команды
	resp, err := m.sendCommand("AT", time.Second)
	if err != nil || !strings.Contains(resp, "OK") {
		return nil
	}

	info := &ModemInfo{
		Port: port,
	}

	// Получаем информацию о производителе
	if resp, err := m.sendCommand("AT+CGMI", time.Second); err == nil {
		info.Manufacturer = extractResponse(resp)
	}

	// Получаем модель
	if resp, err := m.sendCommand("AT+CGMM", time.Second); err == nil {
		info.Model = extractResponse(resp)
	}

	// Получаем IMEI
	if resp, err := m.sendCommand("AT+CGSN", time.Second); err == nil {
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
		port:         serialPort,
		config:       config,
		eventChan:    make(chan Event, 100),
		stopEventsCh: make(chan struct{}),
		reader:       bufio.NewReader(serialPort),
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

	if m.stopEventsCh != nil {
		close(m.stopEventsCh)
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
	timeoutCh := time.After(timeout)

	for {
		select {
		case <-timeoutCh:
			if response.Len() == 0 {
				return "", errors.New("timeout waiting for response")
			}
			return response.String(), nil
		default:
			m.port.SetReadTimeout(time.Millisecond * 100)
			line, err := m.reader.ReadString('\n')
			if err != nil {
				if response.Len() > 0 && strings.Contains(response.String(), "OK") {
					return response.String(), nil
				}
				if response.Len() > 0 && strings.Contains(response.String(), "ERROR") {
					return response.String(), errors.New("command returned ERROR")
				}
				continue
			}

			line = strings.TrimSpace(line)
			if line != "" {
				response.WriteString(line)
				response.WriteString("\n")

				// Проверяем на финальные ответы
				if strings.HasPrefix(line, "OK") ||
					strings.HasPrefix(line, "ERROR") ||
					strings.HasPrefix(line, "+CME ERROR") ||
					strings.HasPrefix(line, "+CMS ERROR") {
					return response.String(), nil
				}
			}
		}
	}
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
func (m *Modem) GetEventChannel() <-chan Event {
	return m.eventChan
}
