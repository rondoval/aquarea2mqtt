package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type aquareaMQTT struct {
	mqttClient     mqtt.Client
	commandChannel chan aquareaCommand
}

func mqttHandler(ctx context.Context, wg *sync.WaitGroup, config configType, dataChannel chan map[string]string, commandChannel chan aquareaCommand, statusChannel chan bool) {
	defer wg.Done()
	log.Println("Starting MQTT handler")
	mqttKeepalive, err := time.ParseDuration(config.MqttKeepalive)
	if err != nil {
		log.Fatal(err)
	}

	var mqttInstance aquareaMQTT
	mqttInstance.commandChannel = commandChannel
	mqttInstance.makeMQTTConn(config.MqttServer, config.MqttPort, config.MqttLogin, config.MqttPass, config.MqttClientID, mqttKeepalive)
	defer mqttInstance.mqttClient.Disconnect(2000)
	defer mqttInstance.setStatus(false)

	for {
		select {
		case dataToPublish := <-dataChannel:
			mqttInstance.publish(dataToPublish)
		case online := <-statusChannel:
			mqttInstance.setStatus(online)
		case <-ctx.Done():
			return
		}
	}
}

func (am *aquareaMQTT) makeMQTTConn(mqttServer string, mqttPort int, mqttLogin, mqttPass, mqttClientID string, mqttKeepalive time.Duration) {
	log.Println("Connecting to MQTT broker")
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

	opts.SetWill("aquarea/status", "offline", byte(0), true)

	// connect to broker
	am.mqttClient = mqtt.NewClient(opts)

	token := am.mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Fail to connect broker, %v", token.Error())
	}
	log.Println("MQTT connected")

	am.setStatus(false) // offline till Service Cloud is connected
}

func (am *aquareaMQTT) setStatus(online bool) {
	var status string
	if online {
		status = "online"
	} else {
		status = "offline"
	}
	am.mqttClient.Publish("aquarea/status", byte(0), true, status)
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
	for key, value := range data {
		token := am.mqttClient.Publish(key, byte(0), true, value)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}
}
