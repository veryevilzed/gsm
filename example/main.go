package main

import (
	"fmt"
	"github.com/veryevilzed/gsm"
	"log"
	"time"
)

func main() {
	// Пример 1: Поиск доступных модемов
	fmt.Println("=== Поиск доступных модемов ===")
	modems, err := gsm.GetAvailableModems()
	if err != nil {
		log.Printf("Ошибка при поиске модемов: %v", err)
	} else {
		for _, modem := range modems {
			fmt.Printf("Найден модем: %s\n", modem.Port)
			fmt.Printf("  Производитель: %s\n", modem.Manufacturer)
			fmt.Printf("  Модель: %s\n", modem.Model)
			fmt.Printf("  IMEI: %s\n", modem.IMEI)
		}
	}

	// Выбираем порт модема (замените на ваш порт)
	port := "/dev/ttyUSB0" // Linux
	// port := "COM3"       // Windows
	// port := "/dev/tty.usbserial0" // macOS

	// Создаем подключение к модему
	modem, err := gsm.New(port, 115200)
	if err != nil {
		log.Fatalf("Не удалось подключиться к модему: %v", err)
	}
	defer modem.Close()

	// Запускаем обработчик событий
	if err := modem.StartEventListener(); err != nil {
		log.Printf("Не удалось запустить обработчик событий: %v", err)
	}

	// Запускаем горутину для обработки событий
	go handleEvents(modem)

	// Пример 2: Получение информации о модеме
	fmt.Println("\n=== Информация о модеме ===")
	info, err := modem.GetExtendedInfo()
	if err != nil {
		log.Printf("Ошибка при получении информации: %v", err)
	} else {
		for key, value := range info {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	// Пример 3: Проверка SIM-карты
	fmt.Println("\n=== Статус SIM-карты ===")
	simStatus, err := modem.GetSIMStatus()
	if err != nil {
		log.Printf("Ошибка при проверке SIM: %v", err)
	} else {
		fmt.Printf("Статус SIM: %s\n", simStatus)

		if simStatus == gsm.PinRequired {
			// Ввод PIN-кода (замените на ваш PIN)
			// err = modem.EnterPIN("1234")
			// if err != nil {
			//     log.Printf("Ошибка при вводе PIN: %v", err)
			// }
		}
	}

	// Пример 4: Проверка сети
	fmt.Println("\n=== Статус сети ===")
	networkStatus, err := modem.GetNetworkStatus()
	if err != nil {
		log.Printf("Ошибка при проверке сети: %v", err)
	} else {
		fmt.Printf("Регистрация в сети: %v\n", networkStatus)
	}

	// Получаем оператора
	operator, err := modem.GetCurrentOperator()
	if err != nil {
		log.Printf("Ошибка при получении оператора: %v", err)
	} else {
		fmt.Printf("Оператор: %s\n", operator.LongName)
	}

	// Получаем качество сигнала
	signal, err := modem.GetSignalQuality()
	if err != nil {
		log.Printf("Ошибка при получении сигнала: %v", err)
	} else {
		fmt.Printf("Уровень сигнала: %d/31 (BER: %d)\n", signal.RSSI, signal.BER)
	}

	// Пример 5: Работа с SMS
	fmt.Println("\n=== Работа с SMS ===")

	// Получаем информацию о хранилище SMS
	storageInfo, err := modem.GetSMSStorageInfo()
	if err != nil {
		log.Printf("Ошибка при получении информации о хранилище: %v", err)
	} else {
		fmt.Printf("SMS в памяти: %s/%s\n", storageInfo["ReadUsed"], storageInfo["ReadTotal"])
	}

	// Чтение всех SMS
	messages, err := modem.ListSMS("ALL")
	if err != nil {
		log.Printf("Ошибка при чтении SMS: %v", err)
	} else {
		fmt.Printf("Найдено сообщений: %d\n", len(messages))
		for _, sms := range messages {
			fmt.Printf("\n[%d] От: %s\n", sms.Index, sms.Sender)
			fmt.Printf("Статус: %s\n", sms.Status)
			fmt.Printf("Время: %s\n", sms.Time.Format("2006-01-02 15:04:05"))
			fmt.Printf("Текст: %s\n", sms.Text)
		}
	}

	// Отправка SMS (раскомментируйте и укажите номер)
	// err = modem.SendSMS("+79991234567", "Тестовое сообщение от GSM модема!")
	// if err != nil {
	//     log.Printf("Ошибка при отправке SMS: %v", err)
	// } else {
	//     fmt.Println("SMS успешно отправлено!")
	// }

	// Пример 6: USSD запросы
	fmt.Println("\n=== USSD запрос ===")
	// Проверка баланса (раскомментируйте для использования)
	// balance, err := modem.SendUSSD("*100#")
	// if err != nil {
	//     log.Printf("Ошибка при отправке USSD: %v", err)
	// } else {
	//     fmt.Printf("Ответ USSD: %s\n", balance)
	// }

	// Пример 7: Поиск операторов (занимает время!)
	// fmt.Println("\n=== Поиск доступных операторов ===")
	// operators, err := modem.ScanOperators()
	// if err != nil {
	//     log.Printf("Ошибка при поиске операторов: %v", err)
	// } else {
	//     for _, op := range operators {
	//         fmt.Printf("Оператор: %s (%s) - %s\n", op.LongName, op.ShortName, op.Numeric)
	//     }
	// }

	// Пример 8: Режимы модема
	fmt.Println("\n=== Режим модема ===")
	mode, err := modem.GetModemMode()
	if err != nil {
		log.Printf("Ошибка при получении режима: %v", err)
	} else {
		fmt.Printf("Текущий режим: %v\n", mode)
	}

	// Ждем немного для демонстрации событий
	fmt.Println("\n=== Ожидание событий (30 секунд) ===")
	fmt.Println("Попробуйте отправить SMS на модем...")
	time.Sleep(30 * time.Second)
}

// handleEvents обрабатывает события от модема
func handleEvents(modem *gsm.Modem) {
	eventChan := modem.GetEventChannel()

	for event := range eventChan {
		fmt.Printf("\n[СОБЫТИЕ] %s в %s\n", event.Type, event.Timestamp.Format("15:04:05"))

		switch event.Type {
		case gsm.EventNewSMS:
			if index, ok := event.Data["index"].(int); ok {
				fmt.Printf("Новое SMS! Индекс: %d\n", index)

				// Автоматически читаем новое сообщение
				sms, err := modem.ReadSMS(index)
				if err != nil {
					log.Printf("Ошибка при чтении SMS: %v", err)
				} else {
					fmt.Printf("От: %s\n", sms.Sender)
					fmt.Printf("Текст: %s\n", sms.Text)
				}
			}

		case gsm.EventIncomingCall:
			if number, ok := event.Data["number"].(string); ok {
				fmt.Printf("Входящий звонок от: %s\n", number)
			} else {
				fmt.Println("Входящий звонок!")
			}

		case gsm.EventNetworkChange:
			if status, ok := event.Data["statusText"].(string); ok {
				fmt.Printf("Изменение статуса сети: %s\n", status)
			}

		case gsm.EventUSSD:
			if msg, ok := event.Data["message"].(string); ok {
				fmt.Printf("USSD ответ: %s\n", msg)
			}

		case gsm.EventModemError:
			if err, ok := event.Data["error"].(string); ok {
				fmt.Printf("Ошибка модема: %s\n", err)
			}
		}
	}
}
