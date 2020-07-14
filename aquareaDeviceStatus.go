package main

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (aq aquarea) parseDeviceStatus(user aquareaEndUserJSON, shiesuahruefutohkun string) (map[string]string, error) {
	r, err := aq.getDeviceStatus(user, shiesuahruefutohkun)
	if err != nil {
		return nil, err
	}
	deviceStatus := make(map[string]string)

	for key, val := range r.StatusDataInfo {
		name := key
		if _, ok := aq.translation[key]; ok {
			name = aq.translation[key].Name
		}
		var value string
		switch val.Type {
		case "basic-text":
			value = aq.dictionaryWebUI[val.TextValue]
		case "simple-value":
			value = val.Value
		}
		deviceStatus[fmt.Sprintf("aquarea/%s/state/%s", user.Gwid, name)] = value

	}
	return deviceStatus, err
}

func (aq *aquarea) getDeviceStatus(user aquareaEndUserJSON, shiesuahruefutohkun string) (aquareaStatusResponseJSON, error) {
	var aquareaStatusResponse aquareaStatusResponseJSON

	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/function/status", url.Values{
		"var.deviceId":        {user.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	})
	if err != nil {
		return aquareaStatusResponse, err

	}
	err = json.Unmarshal(b, &aquareaStatusResponse)
	return aquareaStatusResponse, err
}
