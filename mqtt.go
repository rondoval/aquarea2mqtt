package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func makeMQTTConn() (mqtt.Client, mqtt.Token) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%s", "tcp", config.MqttServer, config.MqttPort))
	opts.SetPassword(config.MqttPass)
	opts.SetUsername(config.MqttLogin)
	opts.SetClientID(config.MqttClientID)

	opts.SetKeepAlive(mqttKeepalive)
	opts.SetOnConnectHandler(startsub)
	opts.SetConnectionLostHandler(connLostHandler)

	// connect to broker
	client := mqtt.NewClient(opts)
	//defer client.Disconnect(uint(2))

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Printf("Fail to connect broker, %v", token.Error())
	}
	return client, token

}

func connLostHandler(c mqtt.Client, err error) {
	log.Printf("Connection lost, reason: %v\n", err)

	//Perform additional action...
}

func startsub(c mqtt.Client) {
	c.Subscribe("aquarea/+/+/set", 2, handleMSGfromMQTT)

	//Perform additional action...
}

func handleMSGfromMQTT(mclient mqtt.Client, msg mqtt.Message) {
	s := strings.Split(msg.Topic(), "/")
	if len(s) > 3 {
		DeviceID := s[1]
		Operation := s[2]
		log.Printf("Device ID %s \n Operation %s", DeviceID, Operation)
		if Operation == "Zone1SetpointTemperature" {
			i, err := strconv.ParseFloat(string(msg.Payload()), 32)
			log.Printf("i=%v, type: %T\n err: %s", i, i, err)
			str := makeChangeHeatingTemperatureJSON(DeviceID, 1, int(i))
			log.Printf("\n %s \n ", str)
			setUserOption(client, DeviceID, str)

		}
	}
	log.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
	log.Printf(".")

}

func publishStates(mclient mqtt.Client, token mqtt.Token, U extractedData) {
	jsonData, err := json.Marshal(U)
	if err != nil {
		fmt.Println("BLAD:", err)
		return
	}
	var m map[string]string
	err = json.Unmarshal([]byte(jsonData), &m)
	if err != nil {
		fmt.Println("BLAD:", err, jsonData)
		return
	}

	for key, value := range m {
		//	fmt.Println("\n", "Key:", key, "Value:", value, "\n")
		TOP := "aquarea/state/" + fmt.Sprintf("%s/%s", m["EnduserID"], key)
		//	fmt.Println("Publikuje do ", TOP, "warosc", value)
		value = strings.TrimSpace(value)
		value = strings.ToUpper(value)
		token = mclient.Publish(TOP, byte(0), false, value)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}

}

func publishLog(mclient mqtt.Client, token mqtt.Token, LD []string, TS int64) {
	TSS := fmt.Sprintf("%d", TS)
	for key, value := range LD {
		//	fmt.Println("\n", "Key:", key, "Value:", value, "\n")
		TOP := "aquarea/log/" + fmt.Sprintf("%d", key)
		fmt.Println("Publikuje do ", TOP, "warosc", value)
		value = strings.TrimSpace(value)
		value = strings.ToUpper(value)
		token = mclient.Publish(TOP, byte(0), false, value)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}
	//	fmt.Println("\n", "Key:", key, "Value:", value, "\n")
	TOP := "aquarea/log/LastUpdated"
	fmt.Println("Publikuje do ", TOP, "warosc", TSS)
	token = mclient.Publish(TOP, byte(0), false, TSS)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish, %v", token.Error())
	}

}
