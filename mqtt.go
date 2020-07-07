package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type aquareaMQTT struct {
	mqttClient     mqtt.Client
	commandChannel chan aquareaCommand
}

func mqttHandler(config configType, dataChannel chan map[string]string, commandChannel chan aquareaCommand) {
	log.Println("Starting MQTT handler")
	mqttKeepalive, err := time.ParseDuration(config.MqttKeepalive)
	if err != nil {
		log.Fatal(err)
	}

	//TODO publish home assistant compatible setup
	var mqttInstance aquareaMQTT
	mqttInstance.commandChannel = commandChannel
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
	opts.SetKeepAlive(mqttKeepalive)

	opts.SetCleanSession(true)  // don't want to receive entire backlog of setting changes
	opts.SetAutoReconnect(true) // default, but I want it explicit
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		c.Subscribe("aquarea/+/settings/+/set", 2, am.handleSubscription)
	})

	// connect to broker
	am.mqttClient = mqtt.NewClient(opts)

	token := am.mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Fail to connect broker, %v", token.Error())
	}
	log.Println("MQTT connected")
}

func (am *aquareaMQTT) handleSubscription(mclient mqtt.Client, msg mqtt.Message) {
	topicPieces := strings.Split(msg.Topic(), "/")
	if len(topicPieces) > 3 {
		deviceID := topicPieces[1]
		setting := topicPieces[3]

		log.Printf("Received: Device ID %s setting: %s", deviceID, setting)
		am.commandChannel <- aquareaCommand{deviceID, setting, string(msg.Payload())}
	}
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
