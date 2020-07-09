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
		log.Println("Dummy value not set")
		return nil
	}
	if len(aq.aquareaSettings.SettingsBackgroundData) == 0 {
		log.Println("Background data not received yet")
		//TODO should we cache the request?
		return nil
	}

	fmt.Println(cmd)
	name := strings.ReplaceAll(aq.reverseTranslation[cmd.setting], "function-setting-user-select-", "userSelect")
	functionInfo := aq.translation[aq.reverseTranslation[cmd.setting]]
	switch functionInfo.Kind {
	case "basic":
		//TODO ugly
		for k, v := range aq.dictionaryWebUI {
			if cmd.value == v {
				cmd.value = k
				break
			}
		}
		for k, v := range functionInfo.Values {
			if cmd.value == v {
				cmd.value = k
				break
			}
		}
	case "placeholder":
		i, _ := strconv.ParseInt(cmd.value, 0, 16)
		if !strings.Contains(cmd.setting, "HolidayMode") {
			//TODO this is not true for all values
			i += 128
		}
		cmd.value = fmt.Sprintf("%X", i)
	}

	user := aq.usersMap[cmd.deviceID]
	shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(user)
	if err != nil {
		return err
	}

	values := url.Values{
		"var.deviceId":        {user.DeviceID},
		"var.preOperation":    {aq.aquareaSettings.SettingsBackgroundData["0x80"].Value},
		"var.preMode":         {aq.aquareaSettings.SettingsBackgroundData["0xE0"].Value},
		"var.preTank":         {aq.aquareaSettings.SettingsBackgroundData["0xE1"].Value},
		"var." + name:         {cmd.value},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	}
	fmt.Println(values)

	fmt.Println("sending")
	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/function/setting/user/set", values)
	fmt.Println("done")
	fmt.Println(string(b))
	fmt.Println(err)
	//TODO usec check api to confirm settings are applied
	return err
}

func (aq *aquarea) receiveSettings(user aquareaEndUserJSON, shiesuahruefutohkun string) (map[string]string, error) {
	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/function/setting/get", url.Values{
		"var.deviceId":        {user.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	})
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &aq.aquareaSettings)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	settings := make(map[string]string)
	settings["EnduserID"] = user.Gwid

	for key, val := range aq.aquareaSettings.SettingDataInfo {
		if !strings.Contains(key, "user") {
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
					//TODO post possible values to a subtopic
					var allOptions string
					for _, option := range translation.Values {
						allOptions += aq.dictionaryWebUI[option] + "\n"
					}
					settings[fmt.Sprintf("settings/%s/options", translation.Name)] = allOptions
				case "placeholder":
					i, _ := strconv.ParseInt(val.SelectedValue, 0, 16)
					if !strings.Contains(translation.Name, "HolidayMode") {
						//TODO this is not true for all values
						i -= 128
					}
					value = fmt.Sprintf("%d", i)
				}
			case "placeholder-text":
				// not used in user settings
				value = val.Placeholder // + val.Params
			}
			settings[fmt.Sprintf("settings/%s", translation.Name)] = value
		}
	}
	return settings, err
}
