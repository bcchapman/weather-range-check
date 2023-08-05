package weatherrange

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/lrosenman/ambient"
	"github.com/sethvargo/go-envconfig"
)

var lastReading float64 = -1.0
var lastReadingInRange bool = false

type (
	deviceFetcher func(ambient.Key) (ambient.APIDeviceResponse, error)
	onChange      func(bool)
)

type MyConfig struct {
	AmbientApplicationKey string `env:"WRC_AMBIENT_APPLICATION_KEY"`
	AmbientAPIKey         string `env:"WRC_AMBIENT_API_KEY"`

	FloorCeiling       float64 `env:"WRC_FLOOR_TEMPERATURE,default=68.0"`
	CeilingTemperature float64 `env:"WRC_CEILING_TEMPERATURE,default=72.0"`

	SleepInterval int `env:"WRC_SLEEP_INTERVAL,default=60"`
}

func StartListening(onChangeHandler onChange) {
	// setup config
	ctx := context.Background()

	var cfg MyConfig
	if err := envconfig.Process(ctx, &cfg); err != nil {
		log.Fatal(err)
	}

	key := ambient.NewKey(cfg.AmbientApplicationKey, cfg.AmbientAPIKey)

	log.Printf("Setting floor temperature to %f and ceiling to %f", cfg.FloorCeiling, cfg.CeilingTemperature)

	for {
		// fetch ambient device
		device, err := fetchDevice(key, ambient.Device)

		if err != nil {
			log.Printf("ERROR: %s\n", err)
		} else {
			// check last reading
			currentReading := device.LastData.Feelslike

			// check if currentReading is in bound
			currentReadingInRange := checkTemperatureInRange(
				currentReading, lastReadingInRange,
				cfg.FloorCeiling, cfg.CeilingTemperature,
				onChangeHandler)

			log.Printf("Previous reading for %s was %.2f, current reading is %.2f\n",
				device.Info.Name,
				lastReading,
				currentReading)

			// assign last reading values
			lastReading = currentReading
			lastReadingInRange = currentReadingInRange
		}
		time.Sleep(time.Duration(cfg.SleepInterval))
	}
}

func fetchDevice(key ambient.Key, fetcher deviceFetcher) (device *ambient.DeviceRecord, err error) {
	devices, ambientError := fetcher(key)

	if ambientError != nil {
		return nil, ambientError
	} else {
		// check for throttling
		if devices.HTTPResponseCode == 429 { // TODO replace with reference to 429
			return nil, errors.New("REQUEST WAS THROTTLED")
		} else if len(devices.DeviceRecord) != 1 { // Check for mismatched device count
			log.Printf("WARNING: Did not receieve expected count of %d device\n", len(devices.DeviceRecord))
			return nil, errors.New("INVALID DEVICE COUNT")
		} else {
			// grab device record
			return &devices.DeviceRecord[0], nil
		}
	}
}

func checkTemperatureInRange(
	curr float64, lastInRange bool, floorTemperature float64,
	ceilingTemperature float64,
	onChangeHandler onChange) bool {

	currentReadingInRange := curr >= floorTemperature && curr <= ceilingTemperature
	if currentReadingInRange != lastInRange {
		log.Printf("Current reading state changed from %v to %v", lastInRange, currentReadingInRange)
		// something changed, send state change
		onChangeHandler(currentReadingInRange)
	}

	return currentReadingInRange
}
