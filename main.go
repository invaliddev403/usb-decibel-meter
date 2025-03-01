package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	hid "github.com/sstallion/go-hid"
)

// Device Info for GM1356
const (
	vendorID  = 25789 // 0x64bd
	productID = 29923 // 0x74e3
)

// GM1356 Commands (must be 8 bytes)
var (
	commandCapture = []byte{0xB3, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // Capture measurement
)

// DecibelReading represents the parsed data from GM1356
type DecibelReading struct {
	Timestamp string  `json:"timestamp"`
	Measured  float64 `json:"measured"`
	Mode      string  `json:"mode"`
	FreqMode  string  `json:"freqMode"`
	Range     string  `json:"range"`
}

// Range mapping based on the C code definition
var rangeMap = map[byte]string{
	0x0: "30-130",
	0x1: "30-80",
	0x2: "50-100",
	0x3: "60-110",
	0x4: "80-130",
}

var logFileName string

func main() {
	// Parse command-line arguments
	flag.StringVar(&logFileName, "log", "", "Specify a CSV file to log measured data")
	flag.Parse()

	// Initialize HIDAPI
	if err := hid.Init(); err != nil {
		log.Fatalf("Failed to initialize HIDAPI: %v", err)
	}
	defer hid.Exit()

	// Open GM1356 Device
	device, err := hid.OpenFirst(vendorID, productID)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer device.Close()
	fmt.Println("Connected to GM1356 Decibel Meter")

	// Open CSV log file if logging is enabled
	var csvFile *os.File
	var csvWriter *csv.Writer
	if logFileName != "" {
		csvFile, csvWriter, err = setupCSVLog(logFileName)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer csvFile.Close()
	}

	// Read current mode, frequency mode, and range before starting measurement
	currentMode, currentFreqMode, currentRange, err := readCurrentMode(device)
	if err != nil {
		log.Printf("Warning: Failed to read current mode. Defaulting to unknown. Error: %v", err)
	} else {
		fmt.Printf("Current Mode: %s, Frequency Mode: %s, Range: %s\n", currentMode, currentFreqMode, currentRange)
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Read data in a separate goroutine
	go readDecibelData(device, stop, csvWriter)

	// Wait for exit signal
	<-stop
	fmt.Println("\nExiting...")
}

// setupCSVLog opens the CSV file for logging and writes headers if the file is new.
func setupCSVLog(filename string) (*os.File, *csv.Writer, error) {
	fileExists := fileExists(filename)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	writer := csv.NewWriter(file)
	if !fileExists {
		// Write CSV header only if the file is new
		writer.Write([]string{"timestamp", "measured", "mode", "freqMode", "range"})
		writer.Flush()
	}
	return file, writer, nil
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// readCurrentMode reads a single packet from the device to determine its mode, frequency mode, and range.
func readCurrentMode(device *hid.Device) (string, string, string, error) {
	buf := make([]byte, 8)

	// Send capture command to request a data sample
	if err := sendCommand(device, commandCapture); err != nil {
		return "unknown", "unknown", "unknown", fmt.Errorf("failed to send initial capture command: %v", err)
	}

	// Read one data packet from the device
	n, err := device.Read(buf)
	if err != nil || n < 6 {
		return "unknown", "unknown", "unknown", fmt.Errorf("failed to read initial data: %v", err)
	}

	// Extract mode, frequency mode, and range
	mode := parseMode(buf[2])
	freqMode := parseFreqMode(buf[2])
	rangeValue := parseRange(buf[2])

	return mode, freqMode, rangeValue, nil
}

// sendCommand sends an 8-byte command to the GM1356
func sendCommand(device *hid.Device, command []byte) error {
	n, err := device.Write(command)
	if err != nil || n != 8 {
		return fmt.Errorf("failed to send command (sent %d bytes): %v", n, err)
	}
	time.Sleep(500 * time.Millisecond) // Wait for device to process command
	fmt.Printf("Command sent: %X\n", command) // Debugging
	return nil
}

// readDecibelData continuously reads and decodes data from the GM1356
func readDecibelData(device *hid.Device, stop chan os.Signal, csvWriter *csv.Writer) {
	buf := make([]byte, 8)

	for {
		select {
		case <-stop:
			return
		default:
			time.Sleep(500 * time.Millisecond) // Prevent excessive polling

			// Send capture command before reading data
			if err := sendCommand(device, commandCapture); err != nil {
				log.Printf("Error sending capture command: %v", err)
				continue
			}

			// Read HID response
			n, err := device.Read(buf)
			if err != nil {
				log.Printf("Error reading data: %v", err)
				continue
			}

			if n > 0 {
				// Debugging: print raw buffer
				fmt.Printf("Raw Data Read (%d bytes): %v\n", n, buf)

				// Parse and print JSON data
				data := parseDecibelData(buf)
				jsonData, _ := json.Marshal(data)
				fmt.Println(string(jsonData))

				// Log data to CSV if enabled
				if csvWriter != nil {
					csvWriter.Write([]string{data.Timestamp, fmt.Sprintf("%.1f", data.Measured), data.Mode, data.FreqMode, data.Range})
					csvWriter.Flush()
				}
			}
		}
	}
}

// parseDecibelData converts raw HID bytes into a structured format
func parseDecibelData(buf []byte) DecibelReading {
	// Extract decibel measurement (16-bit)
	measured := float64((uint16(buf[0]) << 8) | uint16(buf[1])) / 10.0

	// Determine mode, frequency mode, and range
	mode := parseMode(buf[2])
	freqMode := parseFreqMode(buf[2])
	rangeStr := parseRange(buf[2])

	return DecibelReading{
		Measured:  measured,
		Mode:      mode,
		FreqMode:  freqMode,
		Range:     rangeStr,
		Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	}
}

// parseMode decodes fast/slow mode from the HID buffer
func parseMode(b byte) string {
	if b&0x40 != 0 {
		return "fast"
	}
	return "slow"
}

// parseFreqMode decodes dBA/dBC mode from the HID buffer
func parseFreqMode(b byte) string {
	if b&0x10 != 0 || b&0x80 != 0 {
		return "dBC"
	}
	return "dBA"
}

// parseRange extracts the measurement range from the HID buffer
func parseRange(b byte) string {
	if rangeStr, exists := rangeMap[b&0x0F]; exists {
		return rangeStr
	}
	return "unknown"
}
