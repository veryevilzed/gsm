# GSM Modem Library for Go

Библиотека для работы с GSM модемами через последовательный порт (USB to COM) на языке Go. Предоставляет удобную обертку над AT-командами.

## Возможности

- 🔍 Автоматическое обнаружение модемов
- 📱 Отправка и получение SMS
- 📞 Управление звонками
- 🌐 Мониторинг состояния сети
- 📡 USSD запросы
- 🔔 Асинхронная обработка событий
- 🖥️ Поддержка Linux, macOS и Windows

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
// Отправка SMS
err := modem.SendSMS("+79991234567", "Текст сообщения")

// Отправка длинного SMS (автоматическая разбивка)
err := modem.SendLongSMS("+79991234567", "Очень длинное сообщение...")

// Чтение SMS по индексу
sms, _ := modem.ReadSMS(1)

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