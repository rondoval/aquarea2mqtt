package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
)

// Settings panel
func (aq *aquarea) sendSetting(cmd aquareaCommand) error {
	if cmd.value == "----" {
		log.Println("Dummy value - not sending to Aquarea Service Cloud")
		return nil
	}
	if len(aq.aquareaSettings.SettingsBackgroundData) == 0 {
		log.Println("Background data not received yet")
		// should not normally happen - we'll initialize this after log in, before main loop
		return nil
	}

	functionName := aq.reverseTranslation[cmd.setting]
	functionNamePOST := strings.ReplaceAll(functionName, "function-setting-user-select-", "userSelect")
	functionInfo := aq.translation[functionName]

	switch functionInfo.Kind {
	case "basic":
		//reverse translation from friendly name to xxxx-yyyy code and then to hex value
		cmd.value = aq.reverseDictionaryWebUI[cmd.value]
		cmd.value = functionInfo.reverseValues[cmd.value]

	case "placeholder":
		i, _ := strconv.ParseInt(cmd.value, 0, 16)
		if !strings.Contains(cmd.setting, "HolidayMode") {
			// may be not true for all values...
			i += 128
		}
		cmd.value = fmt.Sprintf("0x%X", uint8(i))
	}

	user := aq.usersMap[cmd.deviceID]
	shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(user)
	if err != nil {
		return err
	}

	values := url.Values{
		"var.deviceId":            {user.DeviceID},
		"var.preOperation":        {aq.aquareaSettings.SettingsBackgroundData["0x80"].Value},
		"var.preMode":             {aq.aquareaSettings.SettingsBackgroundData["0xE0"].Value},
		"var.preTank":             {aq.aquareaSettings.SettingsBackgroundData["0xE1"].Value},
		"var." + functionNamePOST: {cmd.value},
		"shiesuahruefutohkun":     {shiesuahruefutohkun},
	}

	log.Printf("Setting %s to %s on %s", cmd.setting, cmd.value, cmd.deviceID)

	_, err = aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/function/setting/user/set", values)
	//TODO parse output json, check error code
	//TODO usec check api to confirm settings are applied before returning
	return err
}

func (aq *aquarea) getDeviceSettings(user aquareaEndUserJSON, shiesuahruefutohkun string) (map[string]string, error) {
	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/function/setting/get", url.Values{
		"var.deviceId":        {user.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	})
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &aq.aquareaSettings)
	if err != nil {
		return nil, err
	}

	settings := make(map[string]string)

	for key, val := range aq.aquareaSettings.SettingDataInfo {
		if !strings.Contains(key, "user") {
			// not an user setting - ignoring
			continue
		}
		if _, ok := aq.translation[key]; ok {
			translation := aq.translation[key]
			var value string
			switch val.Type {
			case "basic-text":
				// not used in user settings
				value = aq.dictionaryWebUI[val.TextValue]
			case "select":
				switch translation.Kind {
				case "basic":
					value = aq.dictionaryWebUI[translation.Values[val.SelectedValue]]

					// post possible values to a subtopic
					var allOptions string
					isFirst := true
					for _, option := range translation.Values {
						if !isFirst {
							allOptions += "\n"
						} else {
							isFirst = false
						}
						allOptions += aq.dictionaryWebUI[option]
					}
					settings[fmt.Sprintf("aquarea/%s/settings/%s/options", user.Gwid, translation.Name)] = allOptions
				case "placeholder":
					i, _ := strconv.ParseInt(val.SelectedValue, 0, 16)
					if !strings.Contains(translation.Name, "HolidayMode") {
						// might be not true for all values...
						i -= 128
					}
					value = strconv.FormatInt(int64(int8(i)), 10) // this is to get two's complement negatives right
				}
			case "placeholder-text":
				// not used in user settings, handling not correct
				value = val.Placeholder // + val.Params
			}
			settings[fmt.Sprintf("aquarea/%s/settings/%s", user.Gwid, translation.Name)] = value
		} else {
			log.Printf("No metadata in translation.json for: %s", key)
		}
	}
	return settings, err
}
