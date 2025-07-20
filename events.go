package gsm

import (
	"fmt"
	"github.com/tarm/serial"
	"strconv"
	"strings"
	"time"
)

// EventType тип события
type EventType string

const (
	EventNewSMS            EventType = "NEW_SMS"
	EventIncomingCall      EventType = "INCOMING_CALL"
	EventCallEnded         EventType = "CALL_ENDED"
	EventNetworkChange     EventType = "NETWORK_CHANGE"
	EventSignalChange      EventType = "SIGNAL_CHANGE"
	EventUSSD              EventType = "USSD"
	EventModemError        EventType = "MODEM_ERROR"
	EventSMSDeliveryReport EventType = "SMS_DELIVERY_REPORT"
)

// Event представляет событие от модема
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      map[string]interface{}
}

// StartEventListener запускает прослушивание событий
func (m *Modem) StartEventListener() error {
	// Настраиваем уведомления о новых SMS
	if err := m.EnableNewSMSNotification(); err != nil {
		return fmt.Errorf("failed to enable SMS notifications: %w", err)
	}

	// Включаем отображение входящих звонков
	if _, err := m.SendCommand("AT+CLIP=1", time.Second); err != nil {
		return fmt.Errorf("failed to enable caller ID: %w", err)
	}

	// Включаем уведомления о изменении регистрации в сети
	if _, err := m.SendCommand("AT+CREG=2", time.Second); err != nil {
		return fmt.Errorf("failed to enable network registration updates: %w", err)
	}

	// Создаем отдельное соединение для событий с коротким таймаутом
	eventConfig := &serial.Config{
		Name:        m.config.Name,
		Baud:        m.config.Baud,
		ReadTimeout: time.Millisecond * 100,
	}

	eventPort, err := serial.OpenPort(eventConfig)
	if err != nil {
		// Используем основной порт если не можем открыть отдельный
		go m.eventListenerLoop()
	} else {
		// Закроем при остановке
		go func() {
			<-m.stopEventsCh
			eventPort.Close()
		}()
		go m.eventListenerLoopWithPort(eventPort)
	}

	return nil
}

// StopEventListener останавливает прослушивание событий
func (m *Modem) StopEventListener() {
	if m.stopEventsCh != nil {
		close(m.stopEventsCh)
	}
}

// eventListenerLoop основной цикл обработки событий
func (m *Modem) eventListenerLoop() {
	buf := make([]byte, 1024)
	var lineBuffer strings.Builder

	for {
		select {
		case <-m.stopEventsCh:
			return
		default:
			// Читаем данные (с таймаутом из конфига)
			n, err := m.port.Read(buf)
			if err != nil {
				// Небольшая пауза при ошибке чтения
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if n > 0 {
				// Добавляем прочитанные данные в буфер
				lineBuffer.Write(buf[:n])

				// Проверяем на наличие полных строк
				content := lineBuffer.String()
				lines := strings.Split(content, "\n")

				// Обрабатываем все полные строки
				for i := 0; i < len(lines)-1; i++ {
					line := strings.TrimSpace(lines[i])
					if line != "" {
						// Обрабатываем событие
						event := m.parseEvent(line)
						if event != nil {
							select {
							case m.eventChan <- *event:
							default:
								// Канал полон, пропускаем событие
							}
						}
					}
				}

				// Оставляем неполную строку в буфере
				lineBuffer.Reset()
				lineBuffer.WriteString(lines[len(lines)-1])
			}
		}
	}
}

// parseEvent парсит строку события
func (m *Modem) parseEvent(line string) *Event {
	event := &Event{
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// Новое SMS
	if strings.HasPrefix(line, "+CMTI:") {
		// +CMTI: "SM",1
		event.Type = EventNewSMS
		parts := strings.Split(line[6:], ",")
		if len(parts) >= 2 {
			event.Data["storage"] = strings.Trim(parts[0], " \"")
			if index, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				event.Data["index"] = index
			}
		}
		return event
	}

	// Входящий звонок
	if strings.HasPrefix(line, "RING") || strings.HasPrefix(line, "+CRING:") {
		event.Type = EventIncomingCall
		return event
	}

	// Информация о звонящем
	if strings.HasPrefix(line, "+CLIP:") {
		// +CLIP: "+79991234567",145,"",,"",0
		event.Type = EventIncomingCall
		parts := strings.Split(line[6:], ",")
		if len(parts) >= 1 {
			event.Data["number"] = strings.Trim(parts[0], " \"")
		}
		return event
	}

	// Изменение регистрации в сети
	if strings.HasPrefix(line, "+CREG:") {
		// +CREG: 2,1,"1234","5678"
		event.Type = EventNetworkChange
		parts := strings.Split(line[6:], ",")
		if len(parts) >= 2 {
			if status, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				event.Data["status"] = NetworkStatus(status)
				event.Data["statusText"] = networkStatusToString(NetworkStatus(status))
			}
		}
		if len(parts) >= 4 {
			event.Data["lac"] = strings.Trim(parts[2], " \"")
			event.Data["cellId"] = strings.Trim(parts[3], " \"")
		}
		return event
	}

	// USSD ответ
	if strings.HasPrefix(line, "+CUSD:") {
		// +CUSD: 0,"Balance: 100.50 RUB",15
		event.Type = EventUSSD
		parts := strings.Split(line[6:], ",")
		if len(parts) >= 2 {
			event.Data["message"] = strings.Trim(parts[1], " \"")
		}
		return event
	}

	// Отчет о доставке SMS
	if strings.HasPrefix(line, "+CDS:") {
		event.Type = EventSMSDeliveryReport
		// Парсим отчет о доставке
		return event
	}

	// Завершение вызова
	if strings.Contains(line, "NO CARRIER") || strings.Contains(line, "BUSY") || strings.Contains(line, "NO ANSWER") {
		event.Type = EventCallEnded
		event.Data["reason"] = line
		return event
	}

	// Ошибки
	if strings.HasPrefix(line, "+CME ERROR:") || strings.HasPrefix(line, "+CMS ERROR:") {
		event.Type = EventModemError
		event.Data["error"] = line
		return event
	}

	// Неизвестное событие - не возвращаем
	return nil
}

// WaitForEvent ждет событие определенного типа с таймаутом
func (m *Modem) WaitForEvent(eventType EventType, timeout time.Duration) (*Event, error) {
	timeoutCh := time.After(timeout)

	for {
		select {
		case event := <-m.eventChan:
			if event.Type == eventType {
				return &event, nil
			}
		case <-timeoutCh:
			return nil, fmt.Errorf("timeout waiting for event %s", eventType)
		}
	}
}

// SendUSSD отправляет USSD запрос
func (m *Modem) SendUSSD(code string) (string, error) {
	// Устанавливаем кодировку для USSD
	if _, err := m.SendCommand("AT+CSCS=\"GSM\"", time.Second); err != nil {
		return "", fmt.Errorf("failed to set encoding: %w", err)
	}

	// Отправляем USSD запрос
	cmd := fmt.Sprintf("AT+CUSD=1,\"%s\",15", code)
	resp, err := m.SendCommand(cmd, time.Second*30)
	if err != nil {
		return "", fmt.Errorf("failed to send USSD: %w", err)
	}

	// Ждем USSD ответ через события
	event, err := m.WaitForEvent(EventUSSD, time.Second*30)
	if err != nil {
		// Пробуем извлечь из прямого ответа
		if strings.Contains(resp, "+CUSD:") {
			parts := strings.Split(resp, ":")
			if len(parts) >= 2 {
				values := strings.Split(parts[1], ",")
				if len(values) >= 2 {
					return strings.Trim(values[1], " \""), nil
				}
			}
		}
		return "", fmt.Errorf("failed to get USSD response: %w", err)
	}

	if msg, ok := event.Data["message"].(string); ok {
		return msg, nil
	}

	return "", fmt.Errorf("invalid USSD response format")
}

// MakeCall совершает звонок
func (m *Modem) MakeCall(number string) error {
	cmd := fmt.Sprintf("ATD%s;", number)
	_, err := m.SendCommand(cmd, time.Second*5)
	if err != nil {
		return fmt.Errorf("failed to make call: %w", err)
	}
	return nil
}

// HangUp завершает текущий вызов
func (m *Modem) HangUp() error {
	_, err := m.SendCommand("ATH", time.Second*2)
	if err != nil {
		return fmt.Errorf("failed to hang up: %w", err)
	}
	return nil
}

// AnswerCall отвечает на входящий вызов
func (m *Modem) AnswerCall() error {
	_, err := m.SendCommand("ATA", time.Second*2)
	if err != nil {
		return fmt.Errorf("failed to answer call: %w", err)
	}
	return nil
}

// SetCallWaiting устанавливает ожидание вызова
func (m *Modem) SetCallWaiting(enable bool) error {
	cmd := "AT+CCWA=0,0"
	if enable {
		cmd = "AT+CCWA=0,1"
	}

	_, err := m.SendCommand(cmd, time.Second*2)
	if err != nil {
		return fmt.Errorf("failed to set call waiting: %w", err)
	}
	return nil
}

// GetCallStatus возвращает статус текущих вызовов
func (m *Modem) GetCallStatus() ([]map[string]string, error) {
	resp, err := m.SendCommand("AT+CLCC", time.Second*2)
	if err != nil {
		return nil, fmt.Errorf("failed to get call status: %w", err)
	}

	var calls []map[string]string
	lines := strings.Split(resp, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "+CLCC:") {
			// +CLCC: 1,0,2,0,0,"+79991234567",145
			call := make(map[string]string)
			parts := strings.Split(line[6:], ",")

			if len(parts) >= 6 {
				call["id"] = strings.TrimSpace(parts[0])
				call["direction"] = mapCallDirection(parts[1])
				call["state"] = mapCallState(parts[2])
				call["mode"] = mapCallMode(parts[3])
				call["multiparty"] = parts[4]
				call["number"] = strings.Trim(parts[5], " \"")
			}

			calls = append(calls, call)
		}
	}

	return calls, nil
}

// Helper functions for call status mapping
func mapCallDirection(dir string) string {
	switch strings.TrimSpace(dir) {
	case "0":
		return "outgoing"
	case "1":
		return "incoming"
	default:
		return "unknown"
	}
}

func mapCallState(state string) string {
	switch strings.TrimSpace(state) {
	case "0":
		return "active"
	case "1":
		return "held"
	case "2":
		return "dialing"
	case "3":
		return "alerting"
	case "4":
		return "incoming"
	case "5":
		return "waiting"
	default:
		return "unknown"
	}
}

func mapCallMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "0":
		return "voice"
	case "1":
		return "data"
	case "2":
		return "fax"
	default:
		return "unknown"
	}
}
