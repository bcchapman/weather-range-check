package weatherrange

import (
	"errors"
	"testing"

	"github.com/lrosenman/ambient"
	"github.com/stretchr/testify/assert"
)

type MockCallback struct {
	wasCalled bool // goLang default bool is false
	inRange   bool
}

func (re *MockCallback) Callback(isTemperatureInRange bool) {
	re.inRange = isTemperatureInRange
	re.wasCalled = true
}

func TestCheckTemperatureInRange(t *testing.T) {
	subtests := []struct {
		name               string
		temperature        float64
		floorTemperature   float64
		ceilingTemperature float64
		expectedResult     bool
		lastInRange        bool
		expectedCallback   bool
	}{
		{
			name:               "Negative temperature",
			temperature:        -12.0,
			floorTemperature:   65.0,
			ceilingTemperature: 73.0,
			expectedResult:     false,
			lastInRange:        false,
			expectedCallback:   false,
		},
		{
			name:               "Less than minimum",
			temperature:        64.0,
			floorTemperature:   65.0,
			ceilingTemperature: 73.0,
			expectedResult:     false,
			lastInRange:        false,
			expectedCallback:   false,
		},
		{
			name:               "Greater than max going out of range",
			temperature:        74.0,
			floorTemperature:   65.0,
			ceilingTemperature: 73.0,
			expectedResult:     false,
			lastInRange:        true,
			expectedCallback:   true,
		},
		{
			name:               "At minimum going into range",
			temperature:        65.0,
			floorTemperature:   65.0,
			ceilingTemperature: 73.0,
			expectedResult:     true,
			lastInRange:        false,
			expectedCallback:   true,
		},
		{
			name:               "At maximum going into range",
			temperature:        73.0,
			floorTemperature:   65.0,
			ceilingTemperature: 73.0,
			expectedResult:     true,
			lastInRange:        false,
			expectedCallback:   true,
		},
	}

	for _, subtest := range subtests {
		t.Run(subtest.name, func(t *testing.T) {
			callback := MockCallback{}
			res := checkTemperatureInRange(
				subtest.temperature, subtest.lastInRange,
				subtest.floorTemperature, subtest.ceilingTemperature,
				callback.Callback)
			if res != subtest.expectedResult {
				t.Errorf("temperature = %.2f: want %t but got %t", subtest.temperature, subtest.expectedResult, res)
			}
			if callback.wasCalled != subtest.expectedCallback {
				t.Errorf("callback called %v but expected %v", callback.wasCalled, subtest.expectedCallback)
			}

			if callback.wasCalled && callback.inRange != res {
				t.Errorf("callback called with %v but expected %v", callback.wasCalled, subtest.expectedCallback)
			}
		})
	}
}

func TestFetchDevice(t *testing.T) {
	mockError := errors.New("UNABLE TO FETCH API RESULTS")
	mockThrottleError := errors.New("REQUEST WAS THROTTLED")
	mockKey := ambient.NewKey("ABC", "DEF")

	subtests := []struct {
		name           string
		deviceFetcher  deviceFetcher
		expectedResult *ambient.DeviceRecord
		expectedErr    error
	}{
		{
			name: "Unable to fetch results",
			deviceFetcher: func(ambient.Key) (response ambient.APIDeviceResponse, err error) {
				return ambient.APIDeviceResponse{}, mockError
			},
			expectedResult: nil,
			expectedErr:    mockError,
		},
		{
			name: "Request was throttled",
			deviceFetcher: func(ambient.Key) (response ambient.APIDeviceResponse, err error) {
				response = ambient.APIDeviceResponse{}
				response.HTTPResponseCode = 429
				return response, nil
			},
			expectedResult: nil,
			expectedErr:    mockThrottleError,
		},
	}

	for _, subtest := range subtests {
		t.Run(subtest.name, func(t *testing.T) {
			res, err := fetchDevice(mockKey, subtest.deviceFetcher)

			assert.EqualErrorf(t, err, subtest.expectedErr.Error(), "Error should be: %v, got: %v", subtest.expectedErr.Error(), err)

			if res != subtest.expectedResult {
				t.Errorf("expected response did not match")
			}
		})
	}
}
