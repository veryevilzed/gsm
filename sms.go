package gsm

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SMS представляет текстовое сообщение
type SMS struct {
	Index    int       // Индекс сообщения в памяти модема (1-255)
	Status   string    // Статус сообщения: "REC UNREAD", "REC READ", "STO SENT", "STO UNSENT"
	Sender   string    // Номер телефона отправителя в международном формате (+7...)
	Receiver string    // Номер телефона получателя (для отправленных сообщений)
	Time     time.Time // Время получения/отправки сообщения
	Text     string    // Текст сообщения (до 160 символов для латиницы, 70 для кириллицы)
}

// SMSStorage представляет хранилище SMS
type SMSStorage string

const (
	StorageSIM       SMSStorage = "SM" // SIM card - хранилище на SIM-карте
	StoragePhone     SMSStorage = "ME" // Phone memory - внутренняя память модема
	StorageAny       SMSStorage = "MT" // Any storage - любое доступное хранилище
	StorageBroadcast SMSStorage = "BM" // Broadcast message - широковещательные сообщения
	StorageStatus    SMSStorage = "SR" // Status report - отчеты о доставке
)

// SendSMS отправляет SMS сообщение
func (m *Modem) SendSMS(number, text string) error {
	// Устанавливаем текстовый режим
	if _, err := m.SendCommand("AT+CMGF=1", time.Second); err != nil {
		return fmt.Errorf("failed to set text mode: %w", err)
	}

	// Проверяем, нужна ли UCS2 кодировка
	needsUCS2 := false
	for _, r := range text {
		if r > 127 {
			needsUCS2 = true
			break
		}
	}

	if needsUCS2 {
		// Устанавливаем UCS2 кодировку
		if _, err := m.SendCommand("AT+CSCS=\"UCS2\"", time.Second); err != nil {
			return fmt.Errorf("failed to set UCS2 encoding: %w", err)
		}

		// Кодируем номер в UCS2
		number = EncodeUCS2(number)
		// Кодируем текст в UCS2
		text = EncodeUCS2(text)
	} else {
		// Устанавливаем GSM кодировку для ASCII
		if _, err := m.SendCommand("AT+CSCS=\"GSM\"", time.Second); err != nil {
			return fmt.Errorf("failed to set GSM encoding: %w", err)
		}
	}

	// Подготавливаем команду отправки
	cmd := fmt.Sprintf("AT+CMGS=\"%s\"", number)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Отправляем команду
	_, err := m.port.Write([]byte(cmd + "\r"))
	if err != nil {
		return fmt.Errorf("failed to initiate SMS: %w", err)
	}

	// Ждем приглашение ">"
	time.Sleep(100 * time.Millisecond)

	// Отправляем текст сообщения с Ctrl+Z (0x1A)
	_, err = m.port.Write([]byte(text + "\x1A"))
	if err != nil {
		return fmt.Errorf("failed to send SMS text: %w", err)
	}

	// Читаем ответ
	resp, err := m.readResponse(time.Second * 30)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	if !strings.Contains(resp, "OK") {
		return fmt.Errorf("SMS sending failed: %s", resp)
	}

	// Возвращаем кодировку обратно на GSM
	m.sendCommand("AT+CSCS=\"GSM\"", time.Second)

	return nil
}

// ReadSMS читает SMS по индексу
func (m *Modem) ReadSMS(index int) (*SMS, error) {
	// Устанавливаем текстовый режим
	if _, err := m.SendCommand("AT+CMGF=1", time.Second); err != nil {
		return nil, fmt.Errorf("failed to set text mode: %w", err)
	}

	// Читаем сообщение
	cmd := fmt.Sprintf("AT+CMGR=%d", index)
	resp, err := m.SendCommand(cmd, time.Second*2)
	if err != nil {
		return nil, fmt.Errorf("failed to read SMS: %w", err)
	}

	return parseSMS(resp, index)
}

// ListSMS возвращает список всех SMS
func (m *Modem) ListSMS(status string) ([]*SMS, error) {
	// Устанавливаем текстовый режим
	if _, err := m.SendCommand("AT+CMGF=1", time.Second); err != nil {
		return nil, fmt.Errorf("failed to set text mode: %w", err)
	}

	// Если статус не указан, читаем все
	if status == "" {
		status = "ALL"
	}

	// Получаем список сообщений
	cmd := fmt.Sprintf("AT+CMGL=\"%s\"", status)
	resp, err := m.SendCommand(cmd, time.Second*5)
	if err != nil {
		return nil, fmt.Errorf("failed to list SMS: %w", err)
	}

	return parseSMSList(resp)
}

// ListUnreadSMS возвращает список непрочитанных SMS
func (m *Modem) ListUnreadSMS() ([]*SMS, error) {
	return m.ListSMS("REC UNREAD")
}

// ListReadSMS возвращает список прочитанных SMS
func (m *Modem) ListReadSMS() ([]*SMS, error) {
	return m.ListSMS("REC READ")
}

// ListSentSMS возвращает список отправленных SMS
func (m *Modem) ListSentSMS() ([]*SMS, error) {
	return m.ListSMS("STO SENT")
}

// ListUnsentSMS возвращает список неотправленных SMS
func (m *Modem) ListUnsentSMS() ([]*SMS, error) {
	return m.ListSMS("STO UNSENT")
}

// DeleteSMS удаляет SMS по индексу
func (m *Modem) DeleteSMS(index int) error {
	cmd := fmt.Sprintf("AT+CMGD=%d", index)
	_, err := m.SendCommand(cmd, time.Second*2)
	if err != nil {
		return fmt.Errorf("failed to delete SMS: %w", err)
	}
	return nil
}

// DeleteAllSMS удаляет все SMS
func (m *Modem) DeleteAllSMS() error {
	// Удаляем все прочитанные сообщения
	_, err := m.SendCommand("AT+CMGD=1,1", time.Second*5)
	if err != nil {
		// Некоторые модемы используют другой синтаксис
		_, err = m.SendCommand("AT+CMGDA=\"DEL ALL\"", time.Second*5)
		if err != nil {
			return fmt.Errorf("failed to delete all SMS: %w", err)
		}
	}
	return nil
}

// CountUnreadSMS возвращает количество непрочитанных SMS
func (m *Modem) CountUnreadSMS() (int, error) {
	messages, err := m.ListUnreadSMS()
	if err != nil {
		return 0, err
	}
	return len(messages), nil
}

// MarkSMSAsRead помечает SMS как прочитанное (читает его)
func (m *Modem) MarkSMSAsRead(index int) error {
	// В GSM модемах сообщение автоматически помечается как прочитанное при чтении
	_, err := m.ReadSMS(index)
	return err
}

// DeleteReadSMS удаляет все прочитанные SMS
func (m *Modem) DeleteReadSMS() error {
	// Удаляем все прочитанные сообщения (флаг 1)
	_, err := m.SendCommand("AT+CMGD=1,1", time.Second*5)
	if err != nil {
		// Альтернативный синтаксис для некоторых модемов
		_, err = m.SendCommand("AT+CMGDA=\"DEL READ\"", time.Second*5)
	}
	return err
}

// DeleteSMSByStatus удаляет SMS по статусу
func (m *Modem) DeleteSMSByStatus(status int) error {
	// status: 0=all received read, 1=all received read, 2=all stored sent, 3=all stored unsent, 4=all messages
	cmd := fmt.Sprintf("AT+CMGD=1,%d", status)
	_, err := m.SendCommand(cmd, time.Second*5)
	return err
}

// SetSMSStorage устанавливает хранилище для SMS
func (m *Modem) SetSMSStorage(readStorage, writeStorage, receiveStorage SMSStorage) error {
	cmd := fmt.Sprintf("AT+CPMS=\"%s\",\"%s\",\"%s\"", readStorage, writeStorage, receiveStorage)
	_, err := m.SendCommand(cmd, time.Second*2)
	if err != nil {
		return fmt.Errorf("failed to set SMS storage: %w", err)
	}
	return nil
}

// GetSMSStorageInfo возвращает информацию о хранилище SMS
func (m *Modem) GetSMSStorageInfo() (map[string]string, error) {
	resp, err := m.SendCommand("AT+CPMS?", time.Second*2)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMS storage info: %w", err)
	}

	info := make(map[string]string)

	// Парсим ответ вида +CPMS: "SM",10,20,"SM",10,20,"SM",10,20
	if strings.Contains(resp, "+CPMS:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			values := strings.Split(strings.TrimSpace(parts[1]), ",")
			if len(values) >= 9 {
				info["ReadStorage"] = strings.Trim(values[0], "\"")
				info["ReadUsed"] = strings.TrimSpace(values[1])
				info["ReadTotal"] = strings.TrimSpace(values[2])
				info["WriteStorage"] = strings.Trim(values[3], "\"")
				info["WriteUsed"] = strings.TrimSpace(values[4])
				info["WriteTotal"] = strings.TrimSpace(values[5])
				info["ReceiveStorage"] = strings.Trim(values[6], "\"")
				info["ReceiveUsed"] = strings.TrimSpace(values[7])
				info["ReceiveTotal"] = strings.TrimSpace(values[8])
			}
		}
	}

	return info, nil
}

// SetNewSMSIndication устанавливает индикацию новых SMS
func (m *Modem) SetNewSMSIndication(mode int, mt int, bm int, ds int, bfr int) error {
	cmd := fmt.Sprintf("AT+CNMI=%d,%d,%d,%d,%d", mode, mt, bm, ds, bfr)
	_, err := m.SendCommand(cmd, time.Second*2)
	if err != nil {
		return fmt.Errorf("failed to set SMS indication: %w", err)
	}
	return nil
}

// EnableNewSMSNotification включает уведомления о новых SMS
func (m *Modem) EnableNewSMSNotification() error {
	// Стандартная настройка: сохранять в память и отправлять уведомление
	return m.SetNewSMSIndication(2, 1, 0, 0, 0)
}

// parseSMS парсит одно SMS сообщение
func parseSMS(response string, index int) (*SMS, error) {
	lines := strings.Split(response, "\n")
	sms := &SMS{Index: index}

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Ищем заголовок сообщения
		if strings.HasPrefix(line, "+CMGR:") {
			// +CMGR: "REC UNREAD","+79991234567","","20/01/01,12:00:00+12"
			parts := strings.Split(line[6:], ",")
			if len(parts) >= 4 {
				sms.Status = strings.Trim(parts[0], " \"")
				sms.Sender = strings.Trim(parts[1], " \"")

				// Парсим время если есть
				if len(parts) >= 4 {
					timeStr := strings.Trim(parts[3], " \"")
					if len(parts) >= 5 {
						timeStr += "," + strings.Trim(parts[4], " \"")
					}
					sms.Time = parseGSMTime(timeStr)
				}
			}

			// Текст сообщения обычно на следующей строке
			if i+1 < len(lines) {
				text := strings.TrimSpace(lines[i+1])
				// Декодируем текст если это UCS2
				sms.Text = DecodeGSMText(text)
			}
			break
		}
	}

	if sms.Status == "" {
		return nil, fmt.Errorf("failed to parse SMS from response")
	}

	return sms, nil
}

// parseSMSList парсит список SMS сообщений
func parseSMSList(response string) ([]*SMS, error) {
	var smsList []*SMS
	lines := strings.Split(response, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Ищем заголовки сообщений
		if strings.HasPrefix(line, "+CMGL:") {
			// +CMGL: 1,"REC UNREAD","+79991234567","","20/01/01,12:00:00+12"
			parts := strings.Split(line[6:], ",")
			if len(parts) >= 2 {
				indexStr := strings.TrimSpace(parts[0])
				index, err := strconv.Atoi(indexStr)
				if err != nil {
					continue
				}

				sms := &SMS{
					Index:  index,
					Status: strings.Trim(parts[1], " \""),
				}

				if len(parts) >= 3 {
					sms.Sender = strings.Trim(parts[2], " \"")
				}

				// Парсим время если есть
				if len(parts) >= 5 {
					timeStr := strings.Trim(parts[4], " \"")
					if len(parts) >= 6 {
						timeStr += "," + strings.Trim(parts[5], " \"")
					}
					sms.Time = parseGSMTime(timeStr)
				}

				// Текст сообщения на следующей строке
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if !strings.HasPrefix(nextLine, "+CMGL:") && nextLine != "OK" && nextLine != "" {
						// Декодируем текст если это UCS2
						sms.Text = DecodeGSMText(nextLine)
						i++ // Пропускаем следующую строку
					}
				}

				smsList = append(smsList, sms)
			}
		}
	}

	return smsList, nil
}

// parseGSMTime парсит время в формате GSM
func parseGSMTime(timeStr string) time.Time {
	// Формат: "20/01/01,12:00:00+12"
	// Разбираем вручную, так как формат специфичный
	parts := strings.Split(timeStr, ",")
	if len(parts) < 2 {
		return time.Time{}
	}

	dateParts := strings.Split(parts[0], "/")
	if len(dateParts) < 3 {
		return time.Time{}
	}

	timeParts := strings.Split(parts[1], "+")
	if len(timeParts) < 1 {
		return time.Time{}
	}

	timeComponents := strings.Split(timeParts[0], ":")
	if len(timeComponents) < 3 {
		return time.Time{}
	}

	// Преобразуем компоненты
	year, _ := strconv.Atoi("20" + dateParts[0])
	month, _ := strconv.Atoi(dateParts[1])
	day, _ := strconv.Atoi(dateParts[2])
	hour, _ := strconv.Atoi(timeComponents[0])
	minute, _ := strconv.Atoi(timeComponents[1])
	second, _ := strconv.Atoi(timeComponents[2])

	// Создаем время (без учета часового пояса для простоты)
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
}

// SendLongSMS отправляет длинное SMS (с разбивкой на части)
func (m *Modem) SendLongSMS(number, text string) error {
	const maxSMSLength = 160

	if len(text) <= maxSMSLength {
		return m.SendSMS(number, text)
	}

	// Для длинных сообщений нужно использовать PDU режим
	// Это упрощенная версия - отправляем по частям
	parts := splitText(text, maxSMSLength-10) // Оставляем место для нумерации

	for i, part := range parts {
		msgText := fmt.Sprintf("[%d/%d] %s", i+1, len(parts), part)
		if err := m.SendSMS(number, msgText); err != nil {
			return fmt.Errorf("failed to send part %d: %w", i+1, err)
		}
		time.Sleep(time.Second) // Пауза между отправками
	}

	return nil
}

// splitText разбивает текст на части
func splitText(text string, maxLen int) []string {
	var parts []string

	for len(text) > 0 {
		if len(text) <= maxLen {
			parts = append(parts, text)
			break
		}

		// Ищем подходящее место для разрыва (пробел)
		splitAt := maxLen
		for i := maxLen - 1; i > maxLen-20 && i > 0; i-- {
			if text[i] == ' ' {
				splitAt = i
				break
			}
		}

		parts = append(parts, text[:splitAt])
		text = strings.TrimSpace(text[splitAt:])
	}

	return parts
}
