package main

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const translationFile = "translation.json"

// for passing MQTT set commands via channel
type aquareaCommand struct {
	deviceID string
	setting  string
	value    string
}
type aquareaFunctionDescription struct {
	Name   string            `json:"name"`
	Kind   string            `json:"kind"`
	Values map[string]string `json:"values"`
}

type aquarea struct {
	AquareaServiceCloudURL      string
	AquareaSmartCloudURL        string
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	logSecOffset                int64
	dataChannel                 chan map[string]string

	httpClient      http.Client
	dictionaryWebUI map[string]string                     // xxxx-yyyy codes translation
	usersMap        map[string]aquareaEndUserJSON         // list of users (devices) linked to an account
	translation     map[string]aquareaFunctionDescription // function name meaning
	logItems        []string                              // table with names of log items (statistics view)
}

func aquareaHandler(config configType, dataChannel chan map[string]string, commandChannel chan aquareaCommand) {
	var aquareaInstance aquarea
	aquareaInstance.AquareaServiceCloudURL = config.AquareaServiceCloudURL
	aquareaInstance.AquareaSmartCloudURL = config.AquareaSmartCloudURL
	aquareaInstance.AquareaServiceCloudLogin = config.AquareaServiceCloudLogin
	aquareaInstance.AquareaServiceCloudPassword = config.AquareaServiceCloudPassword
	aquareaInstance.logSecOffset = config.LogSecOffset
	aquareaInstance.dataChannel = dataChannel
	aquareaInstance.usersMap = make(map[string]aquareaEndUserJSON)

	// Load JSON with translations from Aquarea cryptic names
	data, err := ioutil.ReadFile(translationFile)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &aquareaInstance.translation)
	if err != nil {
		log.Fatal(err)
	}

	poolInterval, err := time.ParseDuration(config.PoolInterval)
	if err != nil {
		log.Fatal(err)
	}

	timeout, err := time.ParseDuration(config.AquareaTimeout)
	if err != nil {
		log.Fatal(err)
	}
	cookieJar, _ := cookiejar.New(nil)
	aquareaInstance.httpClient = http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Jar:       cookieJar,
		Timeout:   timeout,
	}

	for !aquareaInstance.aquareaSetup() {
		//TODO robustness. What if logs out while running
	}
	log.Println("Logged in to Aquarea Service Cloud")

	ticker := time.NewTicker(poolInterval)
	for {
		select {
		case <-ticker.C:
			aquareaInstance.feedDataFromAquarea()
		case command := <-commandChannel:
			aquareaInstance.sendSetting(command)
		}
	}

}

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

func (aq *aquarea) feedDataFromAquarea() {
	for _, user := range aq.usersMap {
		settings, err := aq.testingSettings(user)
		aq.dataChannel <- settings

		// Send device status
		deviceStatus, err := aq.parseDeviceStatus(user)
		if err != nil {
			log.Println(err)
			return
		}
		aq.dataChannel <- deviceStatus

		// Send device logs
		logData, err := aq.getDeviceLogInformation(user)
		if err != nil {
			log.Println(err)
			return
		}
		aq.dataChannel <- logData
	}
}

func (aq aquarea) parseDeviceStatus(user aquareaEndUserJSON) (map[string]string, error) {
	r, err := aq.getDeviceStatus(user)
	deviceStatus := make(map[string]string)
	deviceStatus["EnduserID"] = user.Gwid

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
		deviceStatus[fmt.Sprintf("state/%s", name)] = value

	}
	return deviceStatus, err
}

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

func (aq *aquarea) getShiesuahruefutohkun(url string) (string, error) {
	body, err := aq.httpGet(url)
	if err != nil {
		return "", err
	}
	return aq.extractShiesuahruefutohkun(body)
}

func (aq *aquarea) getEndUserShiesuahruefutohkun(user aquareaEndUserJSON) (string, error) {
	body, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/functionUserInformation", url.Values{
		"var.functionSelectedGwUid": {user.GwUID},
	})
	if err != nil {
		return "", err
	}
	return aq.extractShiesuahruefutohkun(body)
}

func (aq *aquarea) extractShiesuahruefutohkun(body []byte) (string, error) {
	re, err := regexp.Compile(`const shiesuahruefutohkun = '(.+)'`)
	if err != nil {
		return "", err
	}

	ss := re.FindStringSubmatch(string(body))
	if len(ss) > 0 {
		return ss[1], nil
	}
	return "", fmt.Errorf("Could not extract shiesuahruefutohkun")
}

func (aq *aquarea) extractDictionary(body []byte) error {
	dictionaryRegexp, err := regexp.Compile(`const jsonMessage = eval\('\((.+)\)'`)
	dictionaryJSON := dictionaryRegexp.FindStringSubmatch(string(body))
	if len(dictionaryJSON) > 0 {
		result := strings.Replace(dictionaryJSON[1], "\\", "", -1)
		err = json.Unmarshal([]byte(result), &aq.dictionaryWebUI)
	}
	return err
}

func (aq *aquarea) extractLogItems(body []byte) error {
	// We also need a table with statistic names
	logItemsRegexp, err := regexp.Compile(`var logItems = \$\.parseJSON\('(.+)'\);`)
	logItemsJSON := logItemsRegexp.FindStringSubmatch(string(body))
	if len(logItemsJSON) > 0 {
		err = json.Unmarshal([]byte(logItemsJSON[1]), &aq.logItems)
	}

	for key, val := range aq.logItems {
		aq.logItems[key] = strings.ReplaceAll(aq.dictionaryWebUI[val], "/", "\\")
	}
	return err
}

func (aq *aquarea) aquareaLogin() error {
	shiesuahruefutohkun, err := aq.getShiesuahruefutohkun(aq.AquareaServiceCloudURL)
	if err != nil {
		log.Println(err)
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
		log.Println(err)
		return err
	}

	var loginStruct aquareaLoginJSON
	err = json.Unmarshal(b, &loginStruct)

	if loginStruct.ErrorCode != 0 {
		err = fmt.Errorf("%d", loginStruct.ErrorCode)
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
		aq.usersMap[user.GwUID] = user
	}
	aq.getDictionary(endUsersList.Endusers[0])

	return err
}

func (aq *aquarea) getDeviceStatus(user aquareaEndUserJSON) (aquareaStatusResponseJSON, error) {

	var aquareaStatusResponse aquareaStatusResponseJSON
	shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(user)

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

func (aq *aquarea) getDeviceLogInformation(eu aquareaEndUserJSON) (map[string]string, error) {
	shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(eu)
	valueList := "{\"logItems\":["
	for i := range aq.logItems {
		valueList += fmt.Sprintf("%d,", i)
	}
	valueList += "]}"

	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/data/log", url.Values{
		"var.deviceId":        {eu.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
		"var.target":          {"0"},
		"var.startDate":       {fmt.Sprintf("%d000", time.Now().Unix()-aq.logSecOffset)},
		"var.logItems":        {valueList},
	})
	if err != nil {
		return nil, err
	}
	var aquareaLogData aquareaLogDataJSON
	err = json.Unmarshal(b, &aquareaLogData)
	if err != nil {
		return nil, err
	}

	var deviceLog map[int64][]string
	err = json.Unmarshal([]byte(aquareaLogData.LogData), &deviceLog)
	if err != nil {
		return nil, err
	}
	if len(deviceLog) < 1 {
		// no data in log
		return nil, nil
	}

	// we're interested in the most recent snapshot only
	var lastKey int64 = 0
	for k := range deviceLog {
		if lastKey < k {
			lastKey = k
		}
	}

	unitRegexp, err := regexp.Compile(`(.+)\[(.+)\]`)

	stats := make(map[string]string)
	for i, val := range deviceLog[lastKey] {
		split := unitRegexp.FindStringSubmatch(aq.logItems[i])

		topic := "log/" + strings.ReplaceAll(strings.Title(split[1]), " ", "")
		stats[topic+"/unit"] = split[2]
		stats[topic] = val
	}
	stats["log/Timestamp"] = fmt.Sprintf("%d", lastKey)
	stats["log/CurrentError"] = fmt.Sprintf("%d", aquareaLogData.ErrorCode)
	stats["EnduserID"] = eu.Gwid
	return stats, nil
}

// Posts data to Aquarea web service
func (aq *aquarea) httpPost(url string, urlValues url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(urlValues.Encode()))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:74.0) Gecko/20100101 Firefox/74.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := aq.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	return b, err
}

func (aq *aquarea) httpGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:74.0) Gecko/20100101 Firefox/74.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := aq.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	return b, err
}

// Settings panel
func (aq *aquarea) sendSetting(cmd aquareaCommand) error {
	if cmd.value == "----" {
		log.Println("Dummy value not set")
		return nil
	}

	user := aq.usersMap[cmd.deviceID]
	values := make(url.Values)
	values["var.deviceId"] = []string{user.DeviceID}
	//	values["var.preOperation"] = getBackgroundDataValue("0x80")
	//	values["var.preMode"] = getBackgroundDataValue("0xE0")
	//	values["var.preTank"] = getBackgroundDataValue("0xE1")

	// translate from MQTT name to func-set-user-select-xxx
	// translate to userSelectxxx

	//	values["var." + name] = []string{value}

	_, err := aq.getEndUserShiesuahruefutohkun(user)
	if err != nil {
		return err
	}
	_, err = aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/function/setting/user/set", values)

	//TODO usec check api to confirm settings are applied
	return err
}

func (aq *aquarea) testingSettings(user aquareaEndUserJSON) (map[string]string, error) {
	shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(user)
	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/function/setting/get", url.Values{
		"var.deviceId":        {user.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	})
	if err != nil {
		return nil, err
	}
	var aquareaSettings aquareaFunctionSettingGetJSON
	err = json.Unmarshal(b, &aquareaSettings)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	settings := make(map[string]string)
	settings["EnduserID"] = user.Gwid

	for key, val := range aquareaSettings.SettingDataInfo {
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
				case "placeholder":
					i, _ := strconv.ParseInt(val.SelectedValue, 0, 16)
					value = fmt.Sprintf("%d", i-128)
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
