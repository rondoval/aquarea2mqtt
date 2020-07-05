package main

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const configFile = "config.json"

type configType struct {
	AquareaServiceCloudURL      string
	AquareaSmartCloudURL        string
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	AquateaTimeout              int
	MqttServer                  string
	MqttPort                    int
	MqttLogin                   string
	MqttPass                    string
	MqttClientID                string
	MqttKeepalive               string
	PoolInterval                int
	LogSecOffset                int64
}

var aqDevices map[string]enduser

func readConfig() configType {
	var config configType

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

type extractedData struct {
	EnduserID                         string
	RunningStatus                     string
	WorkingMode                       string
	WaterInlet                        string
	WaterOutlet                       string
	Zone1ActualTemperature            string
	Zone1SetpointTemperature          string
	Zone1WaterTemperature             string
	Zone2ActualTemperature            string
	Zone2SetpointTemperature          string
	Zone2WaterTemperature             string
	DailyWaterTankActualTemperature   string
	DailyWaterTankSetpointTemperature string
	BufferTankTemperature             string
	OutdoorTemperature                string
	CompressorStatus                  string
	WaterFlow                         string
	PumpSpeed                         string
	HeatDirection                     string
	RoomHeaterStatus                  string
	DailyWaterHeaterStatus            string
	DefrostStatus                     string
	SolarStatus                       string
	SolarTemperature                  string
	BiMode                            string
	ErrorStatus                       string
	CompressorFrequency               string
	Runtime                           string
	RunCount                          string
	RoomHeaterPerformance             string
	RoomHeaterRunTime                 string
	DailyWaterHeaterRunTime           string
}

func main() {
	config := readConfig()

	dataChannel := make(chan extractedData)
	logChannel := make(chan aquareaLog)

	go mqttHandler(config, dataChannel, logChannel)

	aqDevices = make(map[string]enduser)
	var aquareaInstance aquarea
	aquareaInstance.config = config
	aquareaTimeout := time.Second * time.Duration(config.AquateaTimeout)
	aquareaInstance.poolInterval = time.Second * time.Duration(config.PoolInterval)
	cookieJar, _ := cookiejar.New(nil)
	aquareaInstance.client = http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Jar:       cookieJar,
		Timeout:   aquareaTimeout,
	}

	for {
		aquareaInstance.getAQData()
	}
}
