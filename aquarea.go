package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"errors"
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

type aquarea struct {
	AquareaServiceCloudURL      string
	AquareaSmartCloudURL        string
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	logSecOffset                int64
	dataChannel                 chan map[string]string

	httpClient      http.Client
	dictionaryWebUI map[string]string // xxxx-yyyy codes translation
	usersMap        map[string]aquareaEndUserJSON
	translation     map[string]string // function name meaning
	logItems        []string          // table with names of log items (statistics view)
}

func aquareaHandler(config configType, dataChannel chan map[string]string, commandChannel chan map[string]string) {
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

	for {
		select {
		case command := <-commandChannel:
			if command != nil {
				command = nil
			}
		//TODO handle writes through channel
		default:
			aquareaInstance.feedDataFromAquarea()
			time.Sleep(poolInterval)
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
			name = aq.translation[key]
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

	stats := make(map[string]string)
	for i, val := range deviceLog[lastKey] {
		stats[fmt.Sprintf("log/%s", aq.logItems[i])] = val
	}
	stats["log/timestamp"] = fmt.Sprintf("%d", lastKey)
	stats["log/current_error"] = fmt.Sprintf("%d", aquareaLogData.ErrorCode)
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

func (aq *aquarea) makeChangeHeatingTemperatureJSON(eui string, zoneid int, setpoint int) {
	eu := aq.usersMap[eui]

	setParam := aquareaSetParamJSON{
		Status: []aquareaStatusTypeJSON{
			{
				DeviceGUID: eu.DeviceID,
				ZoneStatus: []aquareaZoneStatusJSON{
					{
						HeatSet: setpoint,
						ZoneID:  zoneid,
					},
				},
			},
		},
	}

	PAYLOAD, err := json.Marshal(setParam)
	if err != nil {
		return
	}
	aq.setUserOption(eui, string(PAYLOAD))
}

// funkcja tylko do testow writow
func (aq *aquarea) setUserOption(eui string, payload string) error {
	eu := aq.usersMap[eui]
	shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(eu)

	var AQCSR aquareaServiceCloudSSOReponseJSON

	_, err = aq.httpClient.Get(aq.AquareaServiceCloudURL + "enduser/confirmStep1Policy")
	CreateSSOUrl := aq.AquareaServiceCloudURL + "/enduser/api/request/create/sso"
	uv := url.Values{
		"var.gwUid":           {eu.GwUID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
	}
	body, err := aq.httpPost(CreateSSOUrl, uv)
	err = json.Unmarshal(body, &AQCSR)
	log.Println(err, body)

	leadInstallerStep1url := aq.AquareaSmartCloudURL + "/remote/leadInstallerStep1"
	uv = url.Values{
		"var.keyCode": {AQCSR.SsoKey},
	}
	_, err = aq.httpPost(leadInstallerStep1url, uv)
	ClaimSSOurl := aq.AquareaSmartCloudURL + "/remote/v1/api/auth/sso"
	uv = url.Values{
		"var.ssoKey": {AQCSR.SsoKey},
	}
	_, err = aq.httpPost(ClaimSSOurl, uv)
	a2wStatusDisplayurl := aq.AquareaSmartCloudURL + "/remote/a2wStatusDisplay"
	uv = url.Values{}
	_, err = aq.httpPost(a2wStatusDisplayurl, uv)
	_, err = aq.httpClient.Get(aq.AquareaSmartCloudURL + "/service-worker.js")
	url := aq.AquareaSmartCloudURL + "/remote/v1/api/devices/" + eu.DeviceID

	//var jsonStr = []byte(`{"status":[{"deviceGuid":"008007B767718332001434545313831373030634345373130434345373138313931304300000000","zoneStatus":[{"zoneId":1,"heatSet":25}]}]}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Referer", aq.AquareaSmartCloudURL+"/remote/a2wControl")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,pl;q=0.8,zh;q=0.7")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", aq.AquareaSmartCloudURL)
	req.Header.Set("Content-Type", "application/json")

	resp, err := aq.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(http.StatusText(resp.StatusCode))
	}
	return nil
}

func (aq *aquarea) testingSettings(user aquareaEndUserJSON) (map[string]string, error) {
	//https://aquarea-service.panasonic.com/installer/functionSetting
	//dictionary - jsonMessage
	// TODO

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
	fmt.Println(aquareaSettings)

	settings := make(map[string]string)
	settings["EnduserID"] = user.Gwid

	for key, val := range aquareaSettings.SettingDataInfo {
		if !strings.Contains(key, "user") {
			continue
		}
		name := key
		if _, ok := aq.translation[key]; ok {
			name = aq.translation[key]
		}
		var value string
		switch val.Type {
		case "basic-text":
			value = aq.dictionaryWebUI[val.TextValue]
		case "select":
			i, _ := strconv.ParseInt(val.SelectedValue, 0, 16)
			if i > 127 {
				i -= 256
			}
			value = fmt.Sprintf("%d", i)
		case "placeholder-text":
			value = val.Placeholder // + val.Params
		}
		settings[fmt.Sprintf("settings/%s", name)] = value
	}
	return settings, err
}
