package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/BurntSushi/toml"
)

const configFile = "config"

var aquareaTimeout time.Duration
var mqttKeepalive time.Duration
var poolInterval time.Duration

var shiesuahruefutohkun string
var lastChecksum [16]byte
var logts int64

type configType struct {
	AquareaServiceCloudURL      string
	AquareaSmartCloudURL        string
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	AquateaTimeout              int
	MqttServer                  string
	MqttPort                    string
	MqttLogin                   string
	MqttPass                    string
	MqttClientID                string
	MqttKeepalive               int
	PoolInterval                int
	LogSecOffset                int64
}

var aqDevices map[string]enduser

func readConfig() configType {
	var config configType
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
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

var client http.Client
var config configType

func main() {
	aqDevices = make(map[string]enduser)

	config = readConfig()
	aquareaTimeout = time.Second * time.Duration(config.AquateaTimeout)
	mqttKeepalive = time.Second * time.Duration(config.MqttKeepalive)
	poolInterval = time.Second * time.Duration(config.PoolInterval)

	cookieJar, _ := cookiejar.New(nil)

	client = http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Jar:       cookieJar,
		Timeout:   aquareaTimeout,
	}
	MC, MT := makeMQTTConn()
	for {
		getAQData(client, MC, MT)
	}
}
