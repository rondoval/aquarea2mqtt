package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"runtime"
)

const configFileOther = "/data/options.json"
const configFileWindows = "options.json"

type configType struct {
	AquareaServiceCloudURL      string
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	AquareaTimeout              string
	PoolInterval                string
	LogSecOffset                int64

	MqttServer    string
	MqttPort      int
	MqttLogin     string
	MqttPass      string
	MqttClientID  string
	MqttKeepalive string
}

func readConfig() configType {
	var config configType
	var configFile string
	if runtime.GOOS == "windows" {
		configFile = configFileWindows
	} else {
		configFile = configFileOther
	}

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

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	config := readConfig()

	dataChannel := make(chan map[string]string, 10)
	commandChannel := make(chan aquareaCommand, 10)
	statusChannel := make(chan bool) // offline-online

	go mqttHandler(config, dataChannel, commandChannel, statusChannel)
	go aquareaHandler(config, dataChannel, commandChannel, statusChannel)

	for {
	}
}
