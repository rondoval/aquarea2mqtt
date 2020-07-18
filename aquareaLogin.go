package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
)

// This gets tus through the entire login process, including populating string translation maps
func (aq *aquarea) aquareaSetup() bool {
	err := aq.aquareaLogin()
	if err != nil {
		log.Println(err)
		return false
	}

	err = aq.aquareaInstallerHome()
	if err != nil {
		log.Println(err)
		return false
	}

	aq.aquareaInitialFetch()

	return true
}

// first fetch of data and Home Assistant discovery
func (aq *aquarea) aquareaInitialFetch() {
	// populate internal data by feeding sub pages
	for _, user := range aq.usersMap {
		// Get settings from the device
		shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(user)
		if err != nil {
			continue
		}

		settings, err := aq.getDeviceSettings(user, shiesuahruefutohkun)
		if err != nil {
			log.Println(err)
		} else {
			// send HA configuration
			haConfig := aq.encodeSwitches(settings, user)
			aq.dataChannel <- haConfig
		}

		_, err = aq.parseDeviceStatus(user, shiesuahruefutohkun)
		if err != nil {
			log.Println(err)
		}
		// not using it for Home Assistant setup - at least for now

		settings, err = aq.getDeviceLogInformation(user, shiesuahruefutohkun)
		if err != nil {
			log.Println(err)
		} else {
			haConfig := aq.encodeSensors(settings, user)
			aq.dataChannel <- haConfig
		}
	}
}

func (aq *aquarea) aquareaLogin() error {
	shiesuahruefutohkun, err := aq.getShiesuahruefutohkun(aq.AquareaServiceCloudURL)
	if err != nil {
		return err
	}

	data := []byte(aq.AquareaServiceCloudLogin + aq.AquareaServiceCloudPassword)
	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"installer/api/auth/login", url.Values{
		"var.loginId":         {aq.AquareaServiceCloudLogin},
		"var.password":        {fmt.Sprintf("%x", md5.Sum(data))},
		"var.inputOmit":       {"false"},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	})
	if err != nil {
		return err
	}

	var loginStruct aquareaLoginJSON
	err = json.Unmarshal(b, &loginStruct)

	if loginStruct.ErrorCode != 0 {
		err = fmt.Errorf("Aquarea login error code: %d", loginStruct.ErrorCode)
	}
	return err
}

func (aq *aquarea) aquareaInstallerHome() error {

	body, err := aq.httpGet(aq.AquareaServiceCloudURL + "installer/home")
	shiesuahruefutohkun, err := aq.extractShiesuahruefutohkun(body)
	if err != nil {
		return err
	}
	err = aq.extractDictionary(body)
	if err != nil {
		return err
	}

	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/endusers", url.Values{
		"var.name":            {""},
		"var.deviceId":        {""},
		"var.idu":             {""},
		"var.odu":             {""},
		"var.sortItem":        {"userName"},
		"var.sortOrder":       {"0"},
		"var.offset":          {"0"},
		"var.limit":           {"999"},
		"var.mapSizeX":        {"0"},
		"var.mapSizeY":        {"0"},
		"var.readNew":         {"1"},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	})
	if err != nil {
		return err
	}
	var endUsersList aquareaEndUsersListJSON
	err = json.Unmarshal(b, &endUsersList)
	if err != nil {
		return err
	}

	for _, user := range endUsersList.Endusers {
		aq.usersMap[user.Gwid] = user
	}
	err = aq.getDictionary(endUsersList.Endusers[0])

	if err == nil {
		aq.statusChannel <- true
	}

	return err
}

// Get lanugage translations from all sub pages
func (aq *aquarea) getDictionary(user aquareaEndUserJSON) error {
	_, err := aq.getEndUserShiesuahruefutohkun(user)
	if err != nil {
		return err
	}

	body, err := aq.httpPost(aq.AquareaServiceCloudURL+"installer/functionSetting", nil)
	if err != nil {
		return err
	}
	err = aq.extractDictionary(body)
	if err != nil {
		return err
	}

	// create reverse dictionary - required for changing settings
	// reverse dictionary is needed for functionSertting page only
	aq.reverseDictionaryWebUI = make(map[string]string)
	for k, v := range aq.dictionaryWebUI {
		aq.reverseDictionaryWebUI[v] = k
	}

	body, err = aq.httpPost(aq.AquareaServiceCloudURL+"installer/functionStatus", nil)
	if err != nil {
		return err
	}
	err = aq.extractDictionary(body)
	if err != nil {
		return err
	}

	body, err = aq.httpPost(aq.AquareaServiceCloudURL+"installer/functionStatistics", nil)
	if err != nil {
		return err
	}
	err = aq.extractDictionary(body)
	if err != nil {
		return err
	}

	err = aq.extractLogItems(body)
	return err
}

func (aq *aquarea) extractDictionary(body []byte) error {
	dictionaryRegexp := regexp.MustCompile(`const jsonMessage = eval\('\((.+)\)'`)
	dictionaryJSON := dictionaryRegexp.FindStringSubmatch(string(body))
	var err error
	if len(dictionaryJSON) > 0 {
		result := strings.Replace(dictionaryJSON[1], "\\", "", -1)
		err = json.Unmarshal([]byte(result), &aq.dictionaryWebUI)
	}
	return err
}

func (aq *aquarea) extractLogItems(body []byte) error {
	// We also need a table with statistic names
	logItemsRegexp := regexp.MustCompile(`var logItems = \$\.parseJSON\('(.+)'\);`)
	logItemsJSON := logItemsRegexp.FindStringSubmatch(string(body))
	var err error
	var items []string
	if len(logItemsJSON) > 0 {
		err = json.Unmarshal([]byte(logItemsJSON[1]), &items)
	}

	unitRegexp := regexp.MustCompile(`(.+)\[(.+)\]`)               // extract unit from name
	unitMultiChoiceRegexp := regexp.MustCompile(`(\d+):([^,]+),?`) // extract multi choice values
	removeBracketsRegexp := regexp.MustCompile(`\(.+\)`)           // remove everything in brackets
	aq.logItems = make([]aquareaLogItem, len(items))

	for key, val := range items {
		val = aq.dictionaryWebUI[val]
		val = strings.ReplaceAll(val, "(Actual)", "Actual")
		val = strings.ReplaceAll(val, "(Target)", "Target")
		val = removeBracketsRegexp.ReplaceAllString(val, "")

		split := unitRegexp.FindStringSubmatch(val)

		name := strings.Title(split[1])
		name = strings.ReplaceAll(name, ":", "")
		name = strings.ReplaceAll(name, " ", "")
		aq.logItems[key].Name = name

		subs := unitMultiChoiceRegexp.FindAllStringSubmatch(split[2], -1)
		aq.logItems[key].Values = make(map[string]string)
		if len(subs) > 0 {
			for _, m := range subs {
				aq.logItems[key].Values[m[1]] = m[2]
			}
		} else {
			aq.logItems[key].Unit = split[2] // unit of the value, extracted from name
		}
	}
	return err
}
