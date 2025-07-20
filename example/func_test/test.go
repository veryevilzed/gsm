package main

import (
	"flag"
	"fmt"
	"github.com/veryevilzed/gsm"
	"log"
	"time"
)

func main() {
	var (
		port     = flag.String("port", "", "Serial port (e.g., /dev/ttyUSB0, COM3)")
		baudRate = flag.Int("baud", 115200, "Baud rate")
		phone    = flag.String("phone", "", "Phone number for SMS test")
		pin      = flag.String("pin", "", "SIM PIN code if required")
	)
	flag.Parse()

	if *port == "" {
		fmt.Println("Searching for available modems...")
		modems, err := gsm.GetAvailableModems()
		if err != nil {
			log.Fatal("Error searching for modems:", err)
		}

		if len(modems) == 0 {
			log.Fatal("No modems found. Please specify port manually with -port flag")
		}

		fmt.Println("\nFound modems:")
		for i, m := range modems {
			fmt.Printf("%d. %s - %s %s (IMEI: %s)\n", i+1, m.Port, m.Manufacturer, m.Model, m.IMEI)
		}

		if len(modems) == 1 {
			*port = modems[0].Port
			fmt.Printf("\nUsing modem at %s\n", *port)
		} else {
			fmt.Print("\nSelect modem number: ")
			var choice int
			fmt.Scanln(&choice)
			if choice < 1 || choice > len(modems) {
				log.Fatal("Invalid choice")
			}
			*port = modems[choice-1].Port
		}
	}

	// Connect to modem
	fmt.Printf("\nConnecting to modem at %s...\n", *port)
	modem, err := gsm.New(*port, *baudRate)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer modem.Close()
	fmt.Println("✓ Connected successfully")

	// События включаем опционально
	fmt.Print("\nEnable event monitoring? (y/n): ")
	var enableEvents string
	fmt.Scanln(&enableEvents)

	if enableEvents == "y" || enableEvents == "Y" {
		if err := modem.StartEventListener(); err != nil {
			log.Printf("Warning: Failed to start event listener: %v", err)
		} else {
			go handleEvents(modem)
			fmt.Println("✓ Event listener started")
		}
	}

	// Run tests
	runTests(modem, *phone, *pin)
}

func runTests(modem *gsm.Modem, testPhone, pin string) {
	fmt.Println("\n=== Running Modem Tests ===\n")

	// Test 1: Connection test
	fmt.Print("1. Testing connection... ")
	if err := modem.TestConnection(); err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
	} else {
		fmt.Println("✓ OK")
	}

	// Test 2: Modem information
	fmt.Println("\n2. Modem Information:")
	if manufacturer, err := modem.GetManufacturer(); err == nil {
		fmt.Printf("   Manufacturer: %s\n", manufacturer)
	}
	if model, err := modem.GetModel(); err == nil {
		fmt.Printf("   Model: %s\n", model)
	}
	if revision, err := modem.GetRevision(); err == nil {
		fmt.Printf("   Revision: %s\n", revision)
	}
	if imei, err := modem.GetIMEI(); err == nil {
		fmt.Printf("   IMEI: %s\n", imei)
	}

	// Test 3: SIM Status
	fmt.Print("\n3. Checking SIM card... ")
	simStatus, err := modem.GetSIMStatus()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
	} else {
		fmt.Printf("Status: %s\n", simStatus)

		if simStatus == gsm.PinRequired && pin != "" {
			fmt.Printf("   Entering PIN... ")
			if err := modem.EnterPIN(pin); err != nil {
				fmt.Printf("✗ Failed: %v\n", err)
			} else {
				fmt.Println("✓ OK")
				time.Sleep(2 * time.Second)
			}
		}
	}

	// Test 4: Network registration
	fmt.Print("\n4. Checking network registration... ")
	netStatus, err := modem.GetNetworkStatus()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
	} else {
		statusStr := "Unknown"
		switch netStatus {
		case gsm.NetworkRegisteredHome:
			statusStr = "Registered (Home)"
		case gsm.NetworkRegisteredRoaming:
			statusStr = "Registered (Roaming)"
		case gsm.NetworkSearching:
			statusStr = "Searching"
		case gsm.NetworkNotRegistered:
			statusStr = "Not Registered"
		case gsm.NetworkRegistrationDenied:
			statusStr = "Registration Denied"
		}
		fmt.Printf("%s\n", statusStr)
	}

	// Test 5: Signal quality
	fmt.Print("\n5. Checking signal quality... ")
	signal, err := modem.GetSignalQuality()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
	} else {
		percentage := (signal.RSSI * 100) / 31
		bars := ""
		if signal.RSSI < 10 {
			bars = "▁"
		} else if signal.RSSI < 15 {
			bars = "▁▂"
		} else if signal.RSSI < 20 {
			bars = "▁▂▃"
		} else if signal.RSSI < 25 {
			bars = "▁▂▃▄"
		} else {
			bars = "▁▂▃▄▅"
		}
		fmt.Printf("RSSI: %d/31 (%d%%) %s\n", signal.RSSI, percentage, bars)
	}

	// Test 6: Current operator
	fmt.Print("\n6. Getting current operator... ")
	operator, err := modem.GetCurrentOperator()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
	} else {
		fmt.Printf("%s\n", operator.LongName)
	}

	// Test 7: SMS Storage
	fmt.Println("\n7. SMS Storage Information:")
	storage, err := modem.GetSMSStorageInfo()
	if err != nil {
		fmt.Printf("   ✗ Failed: %v\n", err)
	} else {
		fmt.Printf("   Read storage: %s (%s/%s used)\n",
			storage["ReadStorage"], storage["ReadUsed"], storage["ReadTotal"])
		fmt.Printf("   Write storage: %s (%s/%s used)\n",
			storage["WriteStorage"], storage["WriteUsed"], storage["WriteTotal"])
	}

	// Test 8: List SMS
	fmt.Print("\n8. Reading SMS messages... ")
	messages, err := modem.ListSMS("ALL")
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
	} else {
		fmt.Printf("Found %d messages\n", len(messages))
		for _, msg := range messages {
			fmt.Printf("   [%d] From: %s, Status: %s\n", msg.Index, msg.Sender, msg.Status)
			fmt.Printf("       Text: %s\n", msg.Text)
		}
	}

	// Test 9: Send SMS (if phone number provided)
	if testPhone != "" {
		fmt.Printf("\n9. Sending test SMS to %s... ", testPhone)
		testMsg := fmt.Sprintf("Test SMS from GSM modem library at %s", time.Now().Format("15:04:05"))
		if err := modem.SendSMS(testPhone, testMsg); err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
		} else {
			fmt.Println("✓ Sent successfully")
		}
	}

	// Test 10: Modem mode
	fmt.Print("\n10. Checking modem mode... ")
	mode, err := modem.GetModemMode()
	if err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
	} else {
		modeStr := "Unknown"
		switch mode {
		case gsm.ModemModeOnline:
			modeStr = "Online (Full functionality)"
		case gsm.ModemModeOffline:
			modeStr = "Offline (Flight mode)"
		case gsm.ModemModeLowPower:
			modeStr = "Low Power"
		}
		fmt.Printf("%s\n", modeStr)
	}

	// Test 11: Phone number (if stored on SIM)
	fmt.Print("\n11. Getting phone number from SIM... ")
	number, err := modem.GetSIMNumber()
	if err != nil {
		fmt.Printf("Not stored on SIM\n")
	} else {
		fmt.Printf("%s\n", number)
	}

	fmt.Println("\n=== Tests completed ===")
	fmt.Println("\nPress Ctrl+C to exit...")
}

func handleEvents(modem *gsm.Modem) {
	events, err := modem.GetEventChannel()
	if err != nil {
		log.Printf("Error getting event channel: %v", err)
		return
	}

	for event := range events {
		fmt.Printf("\n[EVENT] %s at %s\n", event.Type, event.Timestamp.Format("15:04:05"))

		switch event.Type {
		case gsm.EventNewSMS:
			if index, ok := event.Data["index"].(int); ok {
				fmt.Printf("  New SMS received! Index: %d\n", index)

				// Auto-read the message
				if sms, err := modem.ReadSMS(index); err == nil {
					fmt.Printf("  From: %s\n", sms.Sender)
					fmt.Printf("  Text: %s\n", sms.Text)
				}
			}

		case gsm.EventIncomingCall:
			if number, ok := event.Data["number"].(string); ok {
				fmt.Printf("  Incoming call from: %s\n", number)
			}

		case gsm.EventNetworkChange:
			if status, ok := event.Data["statusText"].(string); ok {
				fmt.Printf("  Network status changed: %s\n", status)
			}
		}
	}
}
