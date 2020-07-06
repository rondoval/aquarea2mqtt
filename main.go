package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

const configFile = "config.json"

type configType struct {
	AquareaServiceCloudURL      string
	AquareaSmartCloudURL        string
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
	config := readConfig()

	dataChannel := make(chan map[string]string)
	commandChannel := make(chan map[string]string)

	go mqttHandler(config, dataChannel, commandChannel)
	go aquareaHandler(config, dataChannel, commandChannel)

	for {
	}
}
