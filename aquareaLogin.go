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
	return true
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

	return err
}

// Get lanugage translations from all sub pages
func (aq *aquarea) getDictionary(user aquareaEndUserJSON) error {
	_, err := aq.getEndUserShiesuahruefutohkun(user)
	if err != nil {
		return err
	}
	body, err := aq.httpPost(aq.AquareaServiceCloudURL+"installer/functionStatus", nil)
	if err != nil {
		return err
	}
	err = aq.extractDictionary(body)
	if err != nil {
		return err
	}

	body, err = aq.httpPost(aq.AquareaServiceCloudURL+"installer/functionSetting", nil)
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
	if len(logItemsJSON) > 0 {
		err = json.Unmarshal([]byte(logItemsJSON[1]), &aq.logItems)
	}

	for key, val := range aq.logItems {
		aq.logItems[key] = strings.ReplaceAll(aq.dictionaryWebUI[val], "/", "\\")
	}
	return err
}
