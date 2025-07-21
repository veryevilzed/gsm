# GSM Modem Library for Go

Библиотека для работы с GSM модемами через последовательный порт (USB to COM) на языке Go. Предоставляет удобную обертку над AT-командами.

## Возможности

- 🔍 Автоматическое обнаружение модемов
- 📱 Отправка и получение SMS
- 🌍 Поддержка Unicode/кириллицы в SMS
- 📞 Управление звонками
- 🌐 Мониторинг состояния сети
- 📡 USSD запросы
- 🔔 Асинхронная обработка событий
- 🖥️ Поддержка Linux, macOS и Windows

## Структуры данных

### SMS
Представляет текстовое сообщение:

```go
type SMS struct {
    Index    int       // Индекс сообщения в памяти модема
    Status   string    // Статус: "REC UNREAD", "REC READ", "STO SENT", "STO UNSENT"
    Sender   string    // Номер телефона отправителя (формат: "+79991234567")
    Receiver string    // Номер телефона получателя (для отправленных)
    Time     time.Time // Время получения/отправки сообщения
    Text     string    // Текст сообщения
}
```

### ModemInfo
Информация о модеме:

```go
type ModemInfo struct {
    Port         string // Последовательный порт (например: "/dev/ttyUSB0", "COM3")
    Manufacturer string // Производитель модема (например: "Huawei")
    Model        string // Модель модема (например: "E3372")
    Revision     string // Версия прошивки
    IMEI         string // IMEI модема (15 цифр)
    Description  string // Составное описание: "Manufacturer Model"
}
```

### SignalQuality
Качество сигнала:

```go
type SignalQuality struct {
    RSSI int // Уровень сигнала (0-31, где 31 = отличный, 99 = неизвестно)
    BER  int // Bit Error Rate (0-7, где 0 = отлично, 99 = неизвестно)
}
```

### OperatorInfo
Информация об операторе:

```go
type OperatorInfo struct {
    Status    string // Статус оператора: "0"=неизвестно, "1"=доступен, "2"=текущий, "3"=запрещен
    LongName  string // Полное название (например: "MegaFon")
    ShortName string // Короткое название (например: "MegaFon")
    Numeric   string // Числовой код оператора (например: "25002" для МегаФон)
}
```

### Event
Событие от модема:

```go
type Event struct {
    Type      EventType              // Тип события (см. ниже)
    Timestamp time.Time              // Время события
    Data      map[string]interface{} // Данные события (зависят от типа)
}
```

### Типы событий (EventType)

```go
const (
    EventNewSMS          EventType = "NEW_SMS"          // Новое SMS (Data: "index", "storage")
    EventIncomingCall    EventType = "INCOMING_CALL"    // Входящий звонок (Data: "number")
    EventCallEnded       EventType = "CALL_ENDED"       // Звонок завершен (Data: "reason")
    EventNetworkChange   EventType = "NETWORK_CHANGE"   // Изменение сети (Data: "status", "lac", "cellId")
    EventSignalChange    EventType = "SIGNAL_CHANGE"    // Изменение сигнала
    EventUSSD            EventType = "USSD"             // USSD ответ (Data: "message")
    EventModemError      EventType = "MODEM_ERROR"      // Ошибка модема (Data: "error")
    EventSMSDeliveryReport EventType = "SMS_DELIVERY_REPORT" // Отчет о доставке
)
```

### NetworkStatus
Статус регистрации в сети:

```go
const (
    NetworkNotRegistered      NetworkStatus = 0 // Не зарегистрирован
    NetworkRegisteredHome     NetworkStatus = 1 // Зарегистрирован (домашняя сеть)
    NetworkSearching          NetworkStatus = 2 // Поиск сети
    NetworkRegistrationDenied NetworkStatus = 3 // Регистрация отклонена
    NetworkUnknown            NetworkStatus = 4 // Неизвестный статус
    NetworkRegisteredRoaming  NetworkStatus = 5 // Зарегистрирован (роуминг)
)
```

### PinStatus
Статус PIN-кода:

```go
const (
    PinReady    PinStatus = "READY"     // SIM готова к работе
    PinRequired PinStatus = "SIM PIN"   // Требуется ввод PIN
    PukRequired PinStatus = "SIM PUK"   // Требуется ввод PUK
    PinBlocked  PinStatus = "SIM PIN2"  // PIN2 заблокирован
    PukBlocked  PinStatus = "SIM PUK2"  // PUK2 заблокирован
)
```

### ModemMode
Режим работы модема:

```go
const (
    ModemModeOffline      ModemMode = 0 // Оффлайн режим (режим полета)
    ModemModeOnline       ModemMode = 1 // Полная функциональность
    ModemModeLowPower     ModemMode = 2 // Режим энергосбережения
    ModemModeFactoryTest  ModemMode = 3 // Заводской тест
    ModemModeReset        ModemMode = 4 // Сброс
    ModemModeShuttingDown ModemMode = 5 // Выключение
)
```

### SMSStorage
Хранилище SMS:

```go
const (
    StorageSIM       SMSStorage = "SM" // SIM карта
    StoragePhone     SMSStorage = "ME" // Память телефона/модема
    StorageAny       SMSStorage = "MT" // Любое хранилище
    StorageBroadcast SMSStorage = "BM" // Широковещательные сообщения
    StorageStatus    SMSStorage = "SR" // Отчеты о статусе
)
```

## Примеры использования структур

### Работа с SMS

```go
// Получение SMS возвращает структуру SMS
sms, err := modem.ReadSMS(1)
if err == nil {
    fmt.Printf("От: %s\n", sms.Sender)
    fmt.Printf("Текст: %s\n", sms.Text)
    fmt.Printf("Время: %s\n", sms.Time.Format("02.01.2006 15:04"))
    fmt.Printf("Статус: %s\n", sms.Status)
}

// ListSMS возвращает слайс указателей на SMS
messages, err := modem.ListUnreadSMS()
for _, msg := range messages {
    // msg имеет тип *SMS
    fmt.Printf("[%d] %s: %s\n", msg.Index, msg.Sender, msg.Text)
}
```

### Работа с информацией о модеме

```go
// GetExtendedInfo возвращает map[string]string
info, err := modem.GetExtendedInfo()
if err == nil {
    fmt.Printf("Производитель: %s\n", info["Manufacturer"])
    fmt.Printf("Модель: %s\n", info["Model"])
    fmt.Printf("IMEI: %s\n", info["IMEI"])
    fmt.Printf("Оператор: %s\n", info["Operator"])
    fmt.Printf("Уровень сигнала: %s/31\n", info["SignalRSSI"])
}

// GetSignalQuality возвращает структуру SignalQuality
signal, err := modem.GetSignalQuality()
if err == nil {
    percentage := (signal.RSSI * 100) / 31
    fmt.Printf("Сигнал: %d%% (RSSI: %d, BER: %d)\n", 
        percentage, signal.RSSI, signal.BER)
}
```

### Работа с событиями

```go
// Включаем события
err := modem.StartEventListener()
if err == nil {
    eventChan, _ := modem.GetEventChannel()
    
    for event := range eventChan {
        switch event.Type {
        case gsm.EventNewSMS:
            // Data содержит: index (int), storage (string)
            index := event.Data["index"].(int)
            storage := event.Data["storage"].(string)
            fmt.Printf("Новое SMS #%d в %s\n", index, storage)
            
        case gsm.EventIncomingCall:
            // Data содержит: number (string)
            if number, ok := event.Data["number"].(string); ok {
                fmt.Printf("Звонок от: %s\n", number)
            }
            
        case gsm.EventNetworkChange:
            // Data содержит: status (NetworkStatus), statusText (string), 
            // lac (string), cellId (string)
            status := event.Data["status"].(NetworkStatus)
            fmt.Printf("Сеть изменилась: %v\n", status)
        }
    }
}
```

### Работа с операторами

```go
// GetCurrentOperator возвращает *OperatorInfo
operator, err := modem.GetCurrentOperator()
if err == nil {
    fmt.Printf("Оператор: %s (%s)\n", 
        operator.LongName, operator.Numeric)
}

// ScanOperators возвращает []OperatorInfo
operators, err := modem.ScanOperators()
for _, op := range operators {
    fmt.Printf("%s - %s (код: %s, статус: %s)\n", 
        op.LongName, op.ShortName, op.Numeric, op.Status)
}
```

## Установка

```bash
go get github.com/yourusername/gsm
```

### Зависимости

```bash
go get github.com/tarm/serial
```

## Быстрый старт

```go
package main

import (
    "fmt"
    "log"
    "github.com/yourusername/gsm"
)

func main() {
    // Поиск доступных модемов
    modems, _ := gsm.GetAvailableModems()
    for _, m := range modems {
        fmt.Printf("Найден модем: %s - %s\n", m.Port, m.Description)
    }

    // Подключение к модему
    modem, err := gsm.New("/dev/ttyUSB0", 115200)
    if err != nil {
        log.Fatal(err)
    }
    defer modem.Close()

    // Отправка SMS
    err = modem.SendSMS("+79991234567", "Привет из Go!")
    if err != nil {
        log.Printf("Ошибка отправки SMS: %v", err)
    }
}
```

## Основные функции

### Подключение и информация

```go
// Создание подключения
modem, err := gsm.New(port, baudRate)

// Тест соединения
err := modem.TestConnection()

// Получение информации
manufacturer, _ := modem.GetManufacturer()
model, _ := modem.GetModel()
imei, _ := modem.GetIMEI()
revision, _ := modem.GetRevision()

// Расширенная информация
info, _ := modem.GetExtendedInfo()
```

### Работа с сетью

```go
// Статус регистрации
status, _ := modem.GetNetworkStatus()
gprsStatus, _ := modem.GetGPRSStatus()

// Информация об операторе
operator, _ := modem.GetCurrentOperator()

// Качество сигнала (0-31)
signal, _ := modem.GetSignalQuality()

// Поиск операторов (может занять до 3 минут)
operators, _ := modem.ScanOperators()

// Выбор оператора
err := modem.SelectOperator("25002") // МегаФон
```

### SIM-карта

```go
// Статус SIM
status, _ := modem.GetSIMStatus()

// Ввод PIN-кода
if status == gsm.PinRequired {
    err := modem.EnterPIN("1234")
}

// Получение номера телефона (если сохранен на SIM)
number, _ := modem.GetSIMNumber()
```

### SMS

```go
// Отправка SMS (автоматически определяет кодировку)
err := modem.SendSMS("+79991234567", "Привет!") // Кириллица
err := modem.SendSMS("+79991234567", "Hello!")  // ASCII

// Отправка длинного SMS (автоматическая разбивка)
err := modem.SendLongSMS("+79991234567", "Очень длинное сообщение...")

// Чтение SMS по индексу (автоматическое декодирование)
sms, _ := modem.ReadSMS(1)
fmt.Printf("От: %s\nТекст: %s\n", sms.Sender, sms.Text)

// Список всех SMS
messages, _ := modem.ListSMS("ALL")

// Список непрочитанных SMS
unreadMessages, _ := modem.ListUnreadSMS()

// Список прочитанных SMS
readMessages, _ := modem.ListReadSMS()

// Количество непрочитанных
count, _ := modem.CountUnreadSMS()

// Пометить как прочитанное
err := modem.MarkSMSAsRead(1)

// Удаление SMS
err := modem.DeleteSMS(1)
err := modem.DeleteAllSMS()
err := modem.DeleteReadSMS()

// Настройка хранилища
err := modem.SetSMSStorage(gsm.StorageSIM, gsm.StorageSIM, gsm.StorageSIM)
```

### Работа с Unicode/кириллицей

Библиотека автоматически обрабатывает Unicode текст в SMS:

```go
// Отправка на русском
err := modem.SendSMS("+79991234567", "Привет, как дела?")

// Отправка с emoji
err := modem.SendSMS("+79991234567", "Hello 👋 🚀")

// Смешанный текст
err := modem.SendSMS("+79991234567", "Test тест 测试")

// Ручное декодирование UCS2
decoded, err := gsm.DecodeUCS2("043F044004380432043504420021")
// decoded = "привет!"

// Ручное кодирование в UCS2
encoded := gsm.EncodeUCS2("Привет!")
// encoded = "041F04400438043204350442002100"

// Автоматическое определение и декодирование
text := gsm.DecodeGSMText("043F044004380432043504420021")
// text = "привет!"
```

### Звонки

```go
// Совершить звонок
err := modem.MakeCall("+79991234567")

// Ответить на входящий
err := modem.AnswerCall()

// Завершить вызов
err := modem.HangUp()

// Статус активных вызовов
calls, _ := modem.GetCallStatus()
```

### USSD

```go
// Отправка USSD запроса
response, _ := modem.SendUSSD("*100#") // Проверка баланса
```

### События

События в библиотеке опциональны и должны быть явно включены:

```go
// Проверка статуса обработчика событий
if !modem.IsEventListenerRunning() {
// Запуск обработчика событий
err := modem.StartEventListener()
if err != nil {
log.Printf("Ошибка запуска событий: %v", err)
}
}

// Получение канала событий
eventChan, err := modem.GetEventChannel()
if err != nil {
log.Printf("События не запущены: %v", err)
return
}

// Обработка событий
go func() {
for event := range eventChan {
switch event.Type {
case gsm.EventNewSMS:
index := event.Data["index"].(int)
fmt.Printf("Новое SMS, индекс: %d\n", index)

case gsm.EventIncomingCall:
number := event.Data["number"].(string)
fmt.Printf("Входящий звонок: %s\n", number)
}
}
}()

// Остановка обработчика событий
err = modem.StopEventListener()
```

## Типы событий

- `EventNewSMS` - Новое SMS сообщение
- `EventIncomingCall` - Входящий звонок
- `EventCallEnded` - Завершение вызова
- `EventNetworkChange` - Изменение статуса сети
- `EventSignalChange` - Изменение уровня сигнала
- `EventUSSD` - USSD ответ
- `EventModemError` - Ошибка модема
- `EventSMSDeliveryReport` - Отчет о доставке SMS

## Режимы модема

```go
// Установка режима
modem.SetModemMode(gsm.ModemModeOnline)     // Полный режим
modem.SetModemMode(gsm.ModemModeOffline)    // Режим полёта
modem.SetModemMode(gsm.ModemModeLowPower)   // Минимальный режим

// Получение текущего режима
mode, _ := modem.GetModemMode()
```

## Примеры использования

### Мониторинг входящих SMS

```go
modem.StartEventListener()
events := modem.GetEventChannel()

for event := range events {
if event.Type == gsm.EventNewSMS {
index := event.Data["index"].(int)
sms, _ := modem.ReadSMS(index)

fmt.Printf("SMS от %s: %s\n", sms.Sender, sms.Text)

// Автоответ
modem.SendSMS(sms.Sender, "Сообщение получено!")

// Удаление после обработки
modem.DeleteSMS(index)
}
}
```

### SMS-шлюз

```go
func smsGateway(modem *gsm.Modem) {
// Очистка старых сообщений
    modem.DeleteAllSMS()
    
    // Включение уведомлений
    modem.EnableNewSMSNotification()
    
    // Обработка команд через SMS
    for event := range modem.GetEventChannel() {
        if event.Type == gsm.EventNewSMS {
            sms, _ := modem.ReadSMS(event.Data["index"].(int))
    
            switch sms.Text {
            case "STATUS":
                info, _ := modem.GetExtendedInfo()
                response := fmt.Sprintf("Signal: %s, Operator: %s",
                info["SignalRSSI"], info["Operator"])
                modem.SendSMS(sms.Sender, response)
            
            case "BALANCE":
                balance, _ := modem.SendUSSD("*100#")
                modem.SendSMS(sms.Sender, balance)
            }
    }
}
}
```

## Поддерживаемые модемы

Библиотека работает с большинством GSM модемов, поддерживающих стандартные AT-команды:

- Huawei E173, E3372, E3531
- ZTE MF823, MF831
- Sierra Wireless
- Quectel EC25, M66
- SIMCom SIM800, SIM900
- И другие совместимые модемы

## Требования

- Go 1.16+
- Драйверы для вашего GSM модема
- Активная SIM-карта

## Отладка

Для отладки можно использовать прямую отправку AT-команд:

```go
response, err := modem.SendCommand("AT+COPS?", time.Second*2)
fmt.Println("Ответ:", response)
```

## Лицензия

MIT License

## Вклад в проект

Приветствуются pull requests! Для больших изменений сначала откройте issue для обсуждения.

## TODO

- [ ] Поддержка PDU режима для SMS
- [ ] Работа с контактами SIM-карты
- [ ] Поддержка MMS
- [ ] Работа с несколькими модемами одновременно
- [ ] Более подробная документация AT-команд