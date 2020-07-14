package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type mqttSwitch struct {
	Name         string `json:"name,omitempty"`
	CommandTopic string `json:"command_topic,omitempty"`
	StateTopic   string `json:"state_topic,omitempty"`
	PayloadOn    string `json:"payload_on,omitempty"`
	PayloadOff   string `json:"payload_off,omitempty"`
	UniqueID     string `json:"unique_id,omitempty"`
	Device       struct {
		Manufacturer string `json:"manufacturer,omitempty"`
		Model        string `json:"model,omitempty"`
		Name         string `json:"name,omitempty"`
		Identifiers  string `json:"identifiers,omitempty"`
	} `json:"device"`
}

type mqttSensor struct {
	Name              string `json:"name,omitempty"`
	StateTopic        string `json:"state_topic"`
	UnitOfMeasurement string `json:"unit_of_measurement,omitempty"`
	DeviceClass       string `json:"device_class,omitempty"`
	ForceUpdate       bool   `json:"force_update,omitempty"`
	UniqueID          string `json:"unique_id,omitempty"`
	Device            struct {
		Manufacturer string `json:"manufacturer,omitempty"`
		Model        string `json:"model,omitempty"`
		Name         string `json:"name,omitempty"`
		Identifiers  string `json:"identifiers,omitempty"`
	} `json:"device"`
}

func (aq *aquarea) encodeSwitches(topics map[string]string, user aquareaEndUserJSON) map[string]string {
	config := make(map[string]string)

	for k, v := range topics {
		if strings.Contains(k, "/settings/") && strings.HasSuffix(k, "/options") {
			topicSplit := strings.Split(k, "/")
			name := topicSplit[3]
			deviceID := topicSplit[1]
			values := strings.Split(v, "\n")
			if len(values) <= 2 && len(values) > 0 {
				// 1 or 2 possible values - encode as a switch
				haTopic, haData, err := encodeSwitch(name, deviceID, strings.TrimSuffix(k, "/options"), values)
				if err == nil {
					// send to MQTT
					config[haTopic] = string(haData)
				}
			} else if len(values) > 2 {
				// TODO multi (more than 2) state switch
				// TODO numeric value settings
				// might not be possible
			}
		}
	}

	return config
	//aquarea/B25xxx/settings
	//homeassistant/switch/B2500423423/Operation/config
}

func (aq *aquarea) encodeSensors(topics map[string]string, user aquareaEndUserJSON) map[string]string {
	config := make(map[string]string)

	for k, v := range topics {
		if strings.Contains(k, "/log/") && strings.HasSuffix(k, "/unit") {
			topicSplit := strings.Split(k, "/")
			name := topicSplit[3]
			deviceID := topicSplit[1]

			// v contains the unit
			haTopic, haData, err := encodeSensor(name, deviceID, strings.TrimSuffix(k, "/unit"), v)
			if err == nil {
				// send to MQTT
				config[haTopic] = string(haData)
			}
		}
	}

	return config

	//aquarea/B25xxx/log
	//homeassistant/sensor/B2500423423/Operation/config
}

func encodeSensor(name, id, stateTopic, unit string) (string, []byte, error) {
	var s mqttSensor
	s.Name = name
	s.StateTopic = stateTopic
	s.UnitOfMeasurement = unit
	s.UniqueID = id + "_" + name
	s.Device.Manufacturer = "Panasonic"
	s.Device.Model = "Aquarea"
	s.Device.Identifiers = id
	s.Device.Name = "Aquarea " + id

	//	DeviceClass       string `json:"device_class,omitempty"`
	//	ForceUpdate       bool   `json:"force_update,omitempty"`
	topic := fmt.Sprintf("homeassistant/sensor/%s/%s/config", id, name)
	data, err := json.Marshal(s)

	return topic, data, err
}

func encodeSwitch(name, id, stateTopic string, values []string) (string, []byte, error) {
	var b mqttSwitch
	b.Name = name
	b.CommandTopic = stateTopic + "/set"
	b.StateTopic = stateTopic
	b.Device.Manufacturer = "Panasonic"
	b.Device.Model = "Aquarea"
	b.Device.Identifiers = id
	b.Device.Name = "Aquarea " + id
	b.UniqueID = id + "_" + name

	switchesFound := false
	for _, v := range values {
		if strings.Contains(v, "Off") {
			b.PayloadOff = v
			switchesFound = true
		}
		if strings.Contains(v, "On") {
			b.PayloadOn = v
			switchesFound = true
		}
		if strings.Contains(v, "Request") {
			b.PayloadOn = v
			switchesFound = true
		}
	}

	if !switchesFound {
		return "", nil, fmt.Errorf("Cannot encode switch")
	}

	topic := fmt.Sprintf("homeassistant/switch/%s/%s/config", id, name)
	data, err := json.Marshal(b)

	return topic, data, err
}
