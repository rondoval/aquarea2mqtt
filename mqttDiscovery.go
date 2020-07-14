package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type binarySwitch struct {
	Name         string `json:"name"`
	CommandTopic string `json:"command_topic"`
	StateTopic   string `json:"state_topic"`
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

func (aq *aquarea) encodeSwitches(topics map[string]string, user aquareaEndUserJSON) map[string]string {
	config := make(map[string]string)

	for k, v := range topics {
		if strings.Contains(k, "/settings/") && strings.HasSuffix(k, "/options") {
			topicSplit := strings.Split(k, "/")
			name := topicSplit[3]
			deviceID := topicSplit[1]
			values := strings.Split(v, "\n")
			if len(values) <= 2 {
				haTopic, haData, err := encodeBinarySwitch(name, deviceID, strings.TrimSuffix(k, "/options"), values)
				if err == nil {
					// send to MQTT
					config[haTopic] = string(haData)
				}
			} else {
				// TODO can we encode multi-state switch?
			}
		}
	}

	return config
	//aquarea/B25xxx/state
	//aquarea/B25xxx/settings
	//aquarea/B25xxx/log
	//homeassistant/binary_sensor/B2500423423/Operation/config
}

func encodeBinarySwitch(name, id, stateTopic string, values []string) (string, []byte, error) {
	var b binarySwitch
	b.Name = name
	b.CommandTopic = stateTopic + "/set"
	b.StateTopic = stateTopic
	b.Device.Manufacturer = "Panasonic"
	b.Device.Model = "Aquarea"
	b.Device.Identifiers = id

	for _, v := range values {
		if strings.Contains(v, "Off") {
			b.PayloadOff = v
		}
		if strings.Contains(v, "On") {
			b.PayloadOn = v
		}
	}
	topic := fmt.Sprintf("homeassistant/switch/%s/%s/config", id, name)
	data, err := json.Marshal(b)

	return topic, data, err
}
