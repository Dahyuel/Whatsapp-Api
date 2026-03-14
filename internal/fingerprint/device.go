package fingerprint

import (
	"fmt"
	"math/rand"
)

// DeviceModels is a pool of realistic device models to pick from.
var DeviceModels = []string{
	"Samsung Galaxy S23",
	"Samsung Galaxy S22 Ultra",
	"Google Pixel 8",
	"Google Pixel 7 Pro",
	"OnePlus 12",
	"Xiaomi 13 Pro",
	"iPhone 15 Pro",
	"iPhone 14",
	"OPPO Find X6",
	"Vivo X90 Pro",
}

var osList = []string{"Android", "Android", "Android", "Android", "iOS", "iOS"}
var androidVersions = []string{"13", "14", "12", "11"}
var iosVersions = []string{"17.0", "16.6", "17.2", "15.8"}

// DeviceInfo holds randomized device metadata for a session.
type DeviceInfo struct {
	Model        string
	OSName       string
	OSVersion    string
	AppVersion   string
	Manufacturer string
	BuildNumber  string
}

// NewDeviceInfo generates a random DeviceInfo to use as a session fingerprint.
func NewDeviceInfo() *DeviceInfo {
	model := DeviceModels[rand.Intn(len(DeviceModels))]
	os := osList[rand.Intn(len(osList))]
	var osVer string
	if os == "Android" {
		osVer = androidVersions[rand.Intn(len(androidVersions))]
	} else {
		osVer = iosVersions[rand.Intn(len(iosVersions))]
	}

	manufacturers := map[string]string{
		"Samsung Galaxy S23":    "Samsung",
		"Samsung Galaxy S22 Ultra": "Samsung",
		"Google Pixel 8":        "Google",
		"Google Pixel 7 Pro":    "Google",
		"OnePlus 12":            "OnePlus",
		"Xiaomi 13 Pro":         "Xiaomi",
		"iPhone 15 Pro":         "Apple",
		"iPhone 14":             "Apple",
		"OPPO Find X6":          "OPPO",
		"Vivo X90 Pro":          "Vivo",
	}
	mfr := manufacturers[model]
	if mfr == "" {
		mfr = "Unknown"
	}

	appMajor := 2 + rand.Intn(3)
	appMinor := rand.Intn(40)
	appPatch := rand.Intn(10)

	return &DeviceInfo{
		Model:        model,
		OSName:       os,
		OSVersion:    osVer,
		AppVersion:   fmt.Sprintf("%d.%d.%d", appMajor, appMinor, appPatch),
		Manufacturer: mfr,
		BuildNumber:  fmt.Sprintf("build.%d.%04d", appMajor, rand.Intn(9999)),
	}
}
