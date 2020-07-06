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

func mqttHandler(config configType, dataChannel chan map[string]string, logChannel chan aquareaLog) {
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
			mqttInstance.publishStates(dataToPublish)
		case aquareaLog := <-logChannel:
			mqttInstance.publishLog(aquareaLog)
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
	//TODO more generic one needed, send data to a channel
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

func (am *aquareaMQTT) publishStates(dataToPublish map[string]string) {
	for key, value := range dataToPublish {
		topic := "aquarea/" + fmt.Sprintf("%s/state/%s", dataToPublish["EnduserID"], key)
		value = strings.ToUpper(strings.TrimSpace(value))
		token := am.mqttClient.Publish(topic, byte(0), true, value)
		fmt.Println(topic, ": ", value)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}

}

func (am *aquareaMQTT) publishLog(aqLog aquareaLog) {
	TSS := fmt.Sprintf("%d", aqLog.timestamp)
	for key, value := range aqLog.logData {
		TOP := "aquarea/log/" + fmt.Sprintf("%d", key)
		fmt.Println(TOP, ": ", value)
		value = strings.TrimSpace(value)
		value = strings.ToUpper(value)
		token := am.mqttClient.Publish(TOP, byte(0), false, value)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}
	TOP := "aquarea/log/LastUpdated"
	fmt.Println(TOP, ": ", TSS)
	token := am.mqttClient.Publish(TOP, byte(0), false, TSS)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish, %v", token.Error())
	}
}
