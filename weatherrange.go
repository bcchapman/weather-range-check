package weatherrange

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/lrosenman/ambient"
	"github.com/sethvargo/go-envconfig"
)

const SLEEP_INTERVAL = 15 * time.Second
const THROTTLE_SLEEP_INTERVAL = 30 * time.Second

const FLOOR_TEMPERATURE = 65
const CEILING_TEMPERATURE = 73

var lastReading float64 = -1.0
var lastReadingInRange bool = false

type (
	deviceFetcher func(ambient.Key) (ambient.APIDeviceResponse, error)
	onChange      func(bool)
)

type MyConfig struct {
	AmbientApplicationKey string `env:"AMBIENT_APPLICATION_KEY"`
	AmbientAPIKey         string `env:"AMBIENT_API_KEY"`
}

func StartListening(onChangeHandler onChange) {
	// setup config
	ctx := context.Background()

	var cfg MyConfig
	if err := envconfig.Process(ctx, &cfg); err != nil {
		log.Fatal(err)
	}

	key := ambient.NewKey(cfg.AmbientApplicationKey, cfg.AmbientAPIKey)

	for {
		// fetch ambient device
		device, err := fetchDevice(key, ambient.Device)

		if err != nil {
			log.Printf("ERROR: %s\n", err)
		} else {
			// check last reading
			currentReading := device.LastData.Feelslike

			// check if currentReading is in bound
			currentReadingInRange := checkTemperatureInRange(currentReading, lastReadingInRange, onChangeHandler)

			fmt.Printf("Previous reading for %s was %.2f, current reading is %.2f\n",
				device.Info.Name,
				lastReading,
				currentReading)

			// assign last reading values
			lastReading = currentReading
			lastReadingInRange = currentReadingInRange
		}
		time.Sleep(time.Duration(SLEEP_INTERVAL))
	}
}

func fetchDevice(key ambient.Key, fetcher deviceFetcher) (device *ambient.DeviceRecord, err error) {
	devices, ambientError := fetcher(key)

	if ambientError != nil {
		return nil, ambientError
	} else {
		// check for throttling
		if devices.HTTPResponseCode == 429 { // TODO replace with reference to 429
			// time.Sleep(THROTTLE_SLEEP_INTERVAL) Move to implementer
			return nil, errors.New("REQUEST WAS THROTTLED")
		} else if len(devices.DeviceRecord) != 1 { // Check for mismatched device count
			fmt.Printf("WARNING: Did not receieve expected count of %d device\n", len(devices.DeviceRecord))
			return nil, errors.New("INVALID DEVICE COUNT")
		} else {
			// grab device record
			return &devices.DeviceRecord[0], nil
		}
	}
}

func checkTemperatureInRange(curr float64, lastInRange bool, onChangeHandler onChange) bool {
	currentReadingInRange := curr >= FLOOR_TEMPERATURE && curr <= CEILING_TEMPERATURE
	if currentReadingInRange != lastInRange {
		// something changed, send state change
		onChangeHandler(currentReadingInRange)
	}

	return currentReadingInRange
}
