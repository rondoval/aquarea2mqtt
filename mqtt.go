package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type aquareaMQTT struct {
	mqttClient mqtt.Client
}

func mqttHandler(config configType, dataChannel chan map[string]string, commandChannel chan map[string]string) {
	log.Println("Starting MQTT handler")
	mqttKeepalive, err := time.ParseDuration(config.MqttKeepalive)
	if err != nil {
		log.Fatal(err)
	}

	var mqttInstance aquareaMQTT
	mqttInstance.makeMQTTConn(config.MqttServer, config.MqttPort, config.MqttLogin, config.MqttPass, config.MqttClientID, mqttKeepalive)

	for {
		select {
		case dataToPublish := <-dataChannel:
			mqttInstance.publish(dataToPublish)
		}
	}
}

func (am *aquareaMQTT) makeMQTTConn(mqttServer string, mqttPort int, mqttLogin, mqttPass, mqttClientID string, mqttKeepalive time.Duration) {
	//set MQTT options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%v", "tcp", mqttServer, mqttPort))
	opts.SetPassword(mqttPass)
	opts.SetUsername(mqttLogin)
	opts.SetClientID(mqttClientID)

	opts.SetAutoReconnect(true) // default, but I want it explicit
	opts.SetKeepAlive(mqttKeepalive)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		c.Subscribe("aquarea/+/+/set", 2, handleMSGfromMQTT)
	})

	// connect to broker
	am.mqttClient = mqtt.NewClient(opts)

	token := am.mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Fail to connect broker, %v", token.Error())
	}
	log.Println("MQTT connected")
}

func handleMSGfromMQTT(mclient mqtt.Client, msg mqtt.Message) {
	//TODO more generic one needed, send data to a channel - commandChannel
	s := strings.Split(msg.Topic(), "/")
	if len(s) > 3 {
		DeviceID := s[1]
		Operation := s[2]
		log.Printf("Device ID %s \n Operation %s", DeviceID, Operation)
		if Operation == "Zone1SetpointTemperature" {
			i, err := strconv.ParseFloat(string(msg.Payload()), 32)
			log.Printf("i=%v, type: %T\n err: %s", i, i, err)
			//makeChangeHeatingTemperatureJSON(DeviceID, 1, int(i))
		}
	}
	log.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
}

func (am *aquareaMQTT) publish(data map[string]string) {
	deviceID := data["EnduserID"]
	delete(data, "EnduserID")

	for key, value := range data {
		topic := fmt.Sprintf("aquarea/%s/%s", deviceID, key)
		value = strings.ToUpper(strings.TrimSpace(value))
		fmt.Println(topic, ":", value)

		token := am.mqttClient.Publish(topic, byte(0), true, value)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}
}
