# GM1356 Sound Level Meter Data Logger

## Overview

The **GM1356 Sound Level Meter Data Logger** is a Go-based application that interfaces with the GM1356 sound level meter over HID to read real-time decibel levels, extract measurement metadata (mode, frequency mode, range), and log the data to a JSON-formatted output and optionally a CSV file.

## Features

- **Read real-time decibel levels** from the GM1356 device
- **Identify measurement mode** (Fast/Slow) and frequency mode (dBA/dBC)
- **Determine measurement range** (30-130 dB, 30-80 dB, etc.)
- **Log output to JSON format** in the terminal
- **Optional CSV logging** via `--log` command
- **Graceful shutdown handling** on SIGINT/SIGTERM

## Prerequisites

### Hardware

- GM1356 Sound Level Meter
- USB connection to a computer

### Software

- Go (1.16 or later)
- `github.com/sstallion/go-hid` package
- USB HID permissions (may require `sudo` on Linux)

## Installation

1. Clone this repository:
   ```sh
   git clone https://github.com/invaliddev403/usb-decibel-meter.git
   cd usb-decibel-meter

   ```
2. Install dependencies:
   ```sh
   go mod tidy
   ```
3. Build the application:
   ```sh
   go build -o usb-decibel-meter

   ```

## Usage

### Running the Logger

```sh
go run main.go
```

This will read the decibel levels and print them in JSON format.

### Logging to a CSV File

```sh
go run main.go --log measurements.csv
```

This will append all readings to `measurements.csv` in the following format:

```
timestamp,measured,mode,freqMode,range
2025-03-01 05:04:00 UTC,31.4,fast,dBA,50-100
2025-03-01 05:04:01 UTC,45.3,slow,dBC,30-130
```

### Example Output

```json
{
  "timestamp": "2025-03-01 05:04:00 UTC",
  "measured": 31.4,
  "mode": "fast",
  "freqMode": "dBA",
  "range": "50-100"
}
```

## Permissions (Linux/MacOS)

On some systems, you may need to run the program with `sudo` to access HID devices:

```sh
sudo go run main.go
```

Alternatively, add a **udev rule** to allow non-root users access.

## Troubleshooting

- **Device Not Found:** Ensure the GM1356 is connected and check `dmesg | grep hid` for device detection.
- **Permission Denied:** Run with `sudo` or configure udev rules.
- **Incorrect Readings:** Ensure the correct mode and range settings on the device.

## License

This project is licensed under the MIT License.

## Contributions

Pull requests are welcome! Please open an issue first for major changes.

