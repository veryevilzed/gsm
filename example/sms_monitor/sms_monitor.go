package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/veryevilzed/gsm"
)

const (
	PORT             = "/dev/cu.usbserial-2120" // Измените на ваш порт
	BAUD_RATE        = 9600
	MAX_MESSAGES     = 5
	CHECK_INTERVAL   = 2 * time.Second
	CLEANUP_INTERVAL = 30 * time.Second
)

func main() {
	// Подключаемся к модему
	fmt.Printf("Connecting to modem at %s...\n", PORT)
	modem, err := gsm.New(PORT, BAUD_RATE)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer modem.Close()
	fmt.Println("✓ Connected")

	// Проверяем готовность
	if err := modem.TestConnection(); err != nil {
		log.Fatal("Connection test failed:", err)
	}

	// Запоминаем какие сообщения мы уже видели
	seenMessages := make(map[int]bool)
	var mu sync.Mutex

	// Горутина для проверки новых сообщений
	go func() {
		ticker := time.NewTicker(CHECK_INTERVAL)
		defer ticker.Stop()

		fmt.Println("\n📱 Monitoring SMS... (check every 2 seconds)")
		fmt.Println(strings.Repeat("-", 50))

		for range ticker.C {
			// Получаем непрочитанные
			unread, err := modem.ListUnreadSMS()
			if err != nil {
				continue // Игнорируем ошибки
			}

			// Проверяем каждое сообщение
			for _, sms := range unread {
				mu.Lock()
				if !seenMessages[sms.Index] {
					seenMessages[sms.Index] = true
					mu.Unlock()

					// Показываем новое сообщение
					fmt.Printf("\n🔔 NEW SMS from %s:\n", sms.Sender)
					fmt.Printf("   %s\n", sms.Text)
					fmt.Printf("   [Time: %s, Index: %d]\n",
						sms.Time.Format("15:04:05"), sms.Index)

					// Помечаем как прочитанное
					modem.MarkSMSAsRead(sms.Index)
				} else {
					mu.Unlock()
				}
			}
		}
	}()

	// Горутина для очистки старых сообщений
	go func() {
		ticker := time.NewTicker(CLEANUP_INTERVAL)
		defer ticker.Stop()

		for range ticker.C {
			// Считаем все сообщения
			all, err := modem.ListSMS("ALL")
			if err != nil {
				continue
			}

			// Если больше максимума - чистим
			if len(all) > MAX_MESSAGES {
				fmt.Printf("\n🧹 Cleaning up (total: %d, max: %d)...\n",
					len(all), MAX_MESSAGES)

				// Удаляем старые прочитанные
				read, _ := modem.ListReadSMS()
				deleteCount := len(all) - MAX_MESSAGES
				deleted := 0

				for _, sms := range read {
					if deleted >= deleteCount {
						break
					}
					if modem.DeleteSMS(sms.Index) == nil {
						deleted++
						mu.Lock()
						delete(seenMessages, sms.Index)
						mu.Unlock()
					}
				}

				fmt.Printf("   Deleted %d messages\n", deleted)
			}
		}
	}()

	// Ждем Ctrl+C для выхода
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\n👋 SMS Monitor stopped")
}
