package gsm

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// NetworkStatus представляет статус регистрации в сети
type NetworkStatus int

const (
	NetworkNotRegistered NetworkStatus = iota
	NetworkRegisteredHome
	NetworkSearching
	NetworkRegistrationDenied
	NetworkUnknown
	NetworkRegisteredRoaming
)

// SignalQuality представляет качество сигнала
type SignalQuality struct {
	RSSI int // Received Signal Strength Indicator (0-31, 99=unknown)
	BER  int // Bit Error Rate (0-7, 99=unknown)
}

// OperatorInfo содержит информацию об операторе
type OperatorInfo struct {
	Status    string
	LongName  string
	ShortName string
	Numeric   string
}

// ModemMode представляет режим работы модема
type ModemMode int

const (
	ModemModeOffline ModemMode = iota
	ModemModeOnline
	ModemModeLowPower
	ModemModeFactoryTest
	ModemModeReset
	ModemModeShuttingDown
)

// PinStatus представляет статус PIN-кода
type PinStatus string

const (
	PinReady    PinStatus = "READY"
	PinRequired PinStatus = "SIM PIN"
	PukRequired PinStatus = "SIM PUK"
	PinBlocked  PinStatus = "SIM PIN2"
	PukBlocked  PinStatus = "SIM PUK2"
)

// TestConnection проверяет связь с модемом
func (m *Modem) TestConnection() error {
	resp, err := m.SendCommand("AT", time.Second)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	if !strings.Contains(resp, "OK") {
		return fmt.Errorf("unexpected response: %s", resp)
	}
	return nil
}

// GetManufacturer возвращает производителя модема
func (m *Modem) GetManufacturer() (string, error) {
	resp, err := m.SendCommand("AT+CGMI", time.Second*2)
	if err != nil {
		return "", fmt.Errorf("failed to get manufacturer: %w", err)
	}
	return extractResponse(resp), nil
}

// GetModel возвращает модель модема
func (m *Modem) GetModel() (string, error) {
	resp, err := m.SendCommand("AT+CGMM", time.Second*2)
	if err != nil {
		return "", fmt.Errorf("failed to get model: %w", err)
	}
	return extractResponse(resp), nil
}

// GetRevision возвращает версию прошивки
func (m *Modem) GetRevision() (string, error) {
	resp, err := m.SendCommand("AT+CGMR", time.Second*2)
	if err != nil {
		return "", fmt.Errorf("failed to get revision: %w", err)
	}
	return extractResponse(resp), nil
}

// GetIMEI возвращает IMEI модема
func (m *Modem) GetIMEI() (string, error) {
	resp, err := m.SendCommand("AT+CGSN", time.Second*2)
	if err != nil {
		return "", fmt.Errorf("failed to get IMEI: %w", err)
	}
	return extractResponse(resp), nil
}

// GetNetworkStatus возвращает статус регистрации в сети GSM
func (m *Modem) GetNetworkStatus() (NetworkStatus, error) {
	resp, err := m.SendCommand("AT+CREG?", time.Second*2)
	if err != nil {
		return NetworkUnknown, fmt.Errorf("failed to get network status: %w", err)
	}

	// Парсим ответ вида +CREG: 0,1
	if strings.Contains(resp, "+CREG:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			values := strings.Split(strings.TrimSpace(parts[1]), ",")
			if len(values) >= 2 {
				status, err := strconv.Atoi(strings.TrimSpace(values[1]))
				if err == nil {
					return NetworkStatus(status), nil
				}
			}
		}
	}
	return NetworkUnknown, fmt.Errorf("unexpected response format: %s", resp)
}

// GetGPRSStatus возвращает статус регистрации в GPRS
func (m *Modem) GetGPRSStatus() (NetworkStatus, error) {
	resp, err := m.SendCommand("AT+CGREG?", time.Second*2)
	if err != nil {
		return NetworkUnknown, fmt.Errorf("failed to get GPRS status: %w", err)
	}

	// Парсим ответ вида +CGREG: 0,1
	if strings.Contains(resp, "+CGREG:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			values := strings.Split(strings.TrimSpace(parts[1]), ",")
			if len(values) >= 2 {
				status, err := strconv.Atoi(strings.TrimSpace(values[1]))
				if err == nil {
					return NetworkStatus(status), nil
				}
			}
		}
	}
	return NetworkUnknown, fmt.Errorf("unexpected response format: %s", resp)
}

// GetCurrentOperator возвращает текущего оператора
func (m *Modem) GetCurrentOperator() (*OperatorInfo, error) {
	resp, err := m.SendCommand("AT+COPS?", time.Second*3)
	if err != nil {
		return nil, fmt.Errorf("failed to get current operator: %w", err)
	}

	// Парсим ответ вида +COPS: 0,0,"MegaFon",2
	if strings.Contains(resp, "+COPS:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			values := strings.Split(strings.TrimSpace(parts[1]), ",")
			if len(values) >= 3 {
				operator := &OperatorInfo{
					LongName: strings.Trim(values[2], "\""),
				}
				return operator, nil
			}
		}
	}
	return nil, fmt.Errorf("no operator found or unexpected response: %s", resp)
}

// ScanOperators ищет доступных операторов
func (m *Modem) ScanOperators() ([]OperatorInfo, error) {
	// Это может занять до 3 минут
	resp, err := m.SendCommand("AT+COPS=?", time.Minute*3)
	if err != nil {
		return nil, fmt.Errorf("failed to scan operators: %w", err)
	}

	var operators []OperatorInfo

	// Парсим ответ вида +COPS: (2,"MegaFon","MegaFon","25002",0),...
	if idx := strings.Index(resp, "+COPS:"); idx != -1 {
		resp = resp[idx+6:]
		// Удаляем лишние символы
		resp = strings.TrimSpace(resp)
		resp = strings.Trim(resp, "()")

		// Разделяем операторов
		opStrings := strings.Split(resp, "),(")
		for _, opStr := range opStrings {
			opStr = strings.Trim(opStr, "()")
			parts := strings.Split(opStr, ",")
			if len(parts) >= 4 {
				op := OperatorInfo{
					Status:    strings.TrimSpace(parts[0]),
					LongName:  strings.Trim(parts[1], "\""),
					ShortName: strings.Trim(parts[2], "\""),
					Numeric:   strings.Trim(parts[3], "\""),
				}
				operators = append(operators, op)
			}
		}
	}

	return operators, nil
}

// SelectOperator выбирает оператора
func (m *Modem) SelectOperator(numeric string) error {
	cmd := fmt.Sprintf("AT+COPS=1,2,\"%s\"", numeric)
	_, err := m.SendCommand(cmd, time.Second*30)
	if err != nil {
		return fmt.Errorf("failed to select operator: %w", err)
	}
	return nil
}

// SetAutomaticOperatorSelection устанавливает автоматический выбор оператора
func (m *Modem) SetAutomaticOperatorSelection() error {
	_, err := m.SendCommand("AT+COPS=0", time.Second*30)
	if err != nil {
		return fmt.Errorf("failed to set automatic operator selection: %w", err)
	}
	return nil
}

// GetSignalQuality возвращает качество сигнала
func (m *Modem) GetSignalQuality() (*SignalQuality, error) {
	resp, err := m.SendCommand("AT+CSQ", time.Second*2)
	if err != nil {
		return nil, fmt.Errorf("failed to get signal quality: %w", err)
	}

	// Парсим ответ вида +CSQ: 20,0
	if strings.Contains(resp, "+CSQ:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			values := strings.Split(strings.TrimSpace(parts[1]), ",")
			if len(values) >= 2 {
				rssi, err1 := strconv.Atoi(strings.TrimSpace(values[0]))
				ber, err2 := strconv.Atoi(strings.TrimSpace(values[1]))
				if err1 == nil && err2 == nil {
					return &SignalQuality{
						RSSI: rssi,
						BER:  ber,
					}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("unexpected response format: %s", resp)
}

// GetSIMStatus проверяет статус SIM-карты
func (m *Modem) GetSIMStatus() (PinStatus, error) {
	resp, err := m.SendCommand("AT+CPIN?", time.Second*2)
	if err != nil {
		return "", fmt.Errorf("failed to get SIM status: %w", err)
	}

	// Парсим ответ вида +CPIN: READY
	if strings.Contains(resp, "+CPIN:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			status := strings.TrimSpace(parts[1])
			status = strings.Split(status, "\n")[0] // Убираем OK
			return PinStatus(status), nil
		}
	}
	return "", fmt.Errorf("unexpected response format: %s", resp)
}

// EnterPIN вводит PIN-код
func (m *Modem) EnterPIN(pin string) error {
	cmd := fmt.Sprintf("AT+CPIN=\"%s\"", pin)
	_, err := m.SendCommand(cmd, time.Second*5)
	if err != nil {
		return fmt.Errorf("failed to enter PIN: %w", err)
	}
	return nil
}

// GetSIMNumber пытается получить номер телефона SIM-карты
func (m *Modem) GetSIMNumber() (string, error) {
	resp, err := m.SendCommand("AT+CNUM", time.Second*2)
	if err != nil {
		return "", fmt.Errorf("failed to get SIM number: %w", err)
	}

	// Парсим ответ вида +CNUM: "","79991234567",145
	if strings.Contains(resp, "+CNUM:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			values := strings.Split(strings.TrimSpace(parts[1]), ",")
			if len(values) >= 2 {
				number := strings.Trim(values[1], "\"")
				if number != "" {
					return number, nil
				}
			}
		}
	}
	return "", fmt.Errorf("phone number not stored on SIM")
}

// GetLastFailureReason возвращает последнюю причину отказа регистрации
func (m *Modem) GetLastFailureReason() (string, error) {
	resp, err := m.SendCommand("AT+CEER", time.Second*2)
	if err != nil {
		return "", fmt.Errorf("failed to get failure reason: %w", err)
	}
	return extractResponse(resp), nil
}

// SetModemMode устанавливает режим работы модема
func (m *Modem) SetModemMode(mode ModemMode) error {
	var cmd string
	switch mode {
	case ModemModeOnline:
		cmd = "AT+CFUN=1" // Полный режим
	case ModemModeOffline:
		cmd = "AT+CFUN=4" // Режим полёта
	case ModemModeLowPower:
		cmd = "AT+CFUN=0" // Минимальный режим
	default:
		return fmt.Errorf("unsupported mode: %d", mode)
	}

	_, err := m.SendCommand(cmd, time.Second*10)
	if err != nil {
		return fmt.Errorf("failed to set modem mode: %w", err)
	}
	return nil
}

// GetModemMode возвращает текущий режим модема
func (m *Modem) GetModemMode() (ModemMode, error) {
	resp, err := m.SendCommand("AT+CFUN?", time.Second*2)
	if err != nil {
		return ModemModeOffline, fmt.Errorf("failed to get modem mode: %w", err)
	}

	// Парсим ответ вида +CFUN: 1
	if strings.Contains(resp, "+CFUN:") {
		parts := strings.Split(resp, ":")
		if len(parts) >= 2 {
			modeStr := strings.TrimSpace(parts[1])
			modeStr = strings.Split(modeStr, "\n")[0] // Убираем OK
			mode, err := strconv.Atoi(modeStr)
			if err == nil {
				switch mode {
				case 0:
					return ModemModeLowPower, nil
				case 1:
					return ModemModeOnline, nil
				case 4:
					return ModemModeOffline, nil
				}
			}
		}
	}
	return ModemModeOffline, fmt.Errorf("unexpected response format: %s", resp)
}

// GetExtendedInfo возвращает расширенную информацию о модеме
func (m *Modem) GetExtendedInfo() (map[string]string, error) {
	info := make(map[string]string)

	// Получаем все доступные данные
	if manufacturer, err := m.GetManufacturer(); err == nil {
		info["Manufacturer"] = manufacturer
	}

	if model, err := m.GetModel(); err == nil {
		info["Model"] = model
	}

	if revision, err := m.GetRevision(); err == nil {
		info["Revision"] = revision
	}

	if imei, err := m.GetIMEI(); err == nil {
		info["IMEI"] = imei
	}

	if status, err := m.GetNetworkStatus(); err == nil {
		info["NetworkStatus"] = networkStatusToString(status)
	}

	if signal, err := m.GetSignalQuality(); err == nil {
		info["SignalRSSI"] = fmt.Sprintf("%d", signal.RSSI)
		info["SignalBER"] = fmt.Sprintf("%d", signal.BER)
	}

	if operator, err := m.GetCurrentOperator(); err == nil {
		info["Operator"] = operator.LongName
	}

	if simStatus, err := m.GetSIMStatus(); err == nil {
		info["SIMStatus"] = string(simStatus)
	}

	return info, nil
}

// networkStatusToString преобразует NetworkStatus в строку
func networkStatusToString(status NetworkStatus) string {
	switch status {
	case NetworkNotRegistered:
		return "Not registered"
	case NetworkRegisteredHome:
		return "Registered (home)"
	case NetworkSearching:
		return "Searching"
	case NetworkRegistrationDenied:
		return "Registration denied"
	case NetworkUnknown:
		return "Unknown"
	case NetworkRegisteredRoaming:
		return "Registered (roaming)"
	default:
		return "Unknown"
	}
}
