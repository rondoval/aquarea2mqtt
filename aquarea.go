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
	"sort"
	"strings"
	"time"
)

type aquarea struct {
	AquareaServiceCloudURL      string
	AquareaSmartCloudURL        string
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	logSecOffset                int64
	dataChannel                 chan aquareaDeviceStatus
	logChannel                  chan aquareaLog

	httpClient      http.Client
	lastChecksum    [16]byte
	logTimestamp    int64
	dictionaryWebUI map[string]string
	usersMap        map[string]endUserJSON
}

type aquareaLog struct {
	logData   []string
	timestamp int64
}

type aquareaDeviceStatus struct {
	EnduserID                         string
	RunningStatus                     string
	WorkingMode                       string
	WaterInlet                        string
	WaterOutlet                       string
	Zone1ActualTemperature            string
	Zone1SetpointTemperature          string
	Zone1WaterTemperature             string
	Zone2ActualTemperature            string
	Zone2SetpointTemperature          string
	Zone2WaterTemperature             string
	DailyWaterTankActualTemperature   string
	DailyWaterTankSetpointTemperature string
	BufferTankTemperature             string
	OutdoorTemperature                string
	CompressorStatus                  string
	WaterFlow                         string
	PumpSpeed                         string
	HeatDirection                     string
	RoomHeaterStatus                  string
	DailyWaterHeaterStatus            string
	DefrostStatus                     string
	SolarStatus                       string
	SolarTemperature                  string
	BiMode                            string
	ErrorStatus                       string
	CompressorFrequency               string
	Runtime                           string
	RunCount                          string
	RoomHeaterPerformance             string
	RoomHeaterRunTime                 string
	DailyWaterHeaterRunTime           string
}

func aquareaHandler(config configType, dataChannel chan aquareaDeviceStatus, logChannel chan aquareaLog) {
	var aquareaInstance aquarea
	aquareaInstance.AquareaServiceCloudURL = config.AquareaServiceCloudURL
	aquareaInstance.AquareaSmartCloudURL = config.AquareaSmartCloudURL
	aquareaInstance.AquareaServiceCloudLogin = config.AquareaServiceCloudLogin
	aquareaInstance.AquareaServiceCloudPassword = config.AquareaServiceCloudPassword
	aquareaInstance.logSecOffset = config.LogSecOffset
	aquareaInstance.dataChannel = dataChannel
	aquareaInstance.logChannel = logChannel
	aquareaInstance.usersMap = make(map[string]endUserJSON)

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

	for {
		select {
		//TODO handle writes through channel
		default:
			aquareaInstance.parseAllDevices()
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

func (aq *aquarea) parseAllDevices() {
	for _, user := range aq.usersMap {
		// Send device status
		deviceStatus, err := aq.parseDevice(user)
		if err != nil {
			log.Println(err)
			return
		}
		md5 := md5.Sum([]byte(fmt.Sprintf("%s", deviceStatus)))

		if md5 != aq.lastChecksum {
			aq.dataChannel <- deviceStatus
			aq.lastChecksum = md5
		}

		// Send device logs
		logData, err := aq.getDeviceLogInformation(user)
		if err != nil {
			log.Println(err)
			return
		}

		if logData.timestamp != aq.logTimestamp {
			aq.logChannel <- logData
			aq.logTimestamp = logData.timestamp
		}
	}
}

func (aq aquarea) parseDevice(user endUserJSON) (aquareaDeviceStatus, error) {
	r, err := aq.getDeviceStatus(user)

	var deviceStatus aquareaDeviceStatus
	deviceStatus.EnduserID = user.Gwid
	deviceStatus.RunningStatus = aq.translateCodeToString(r.StatusDataInfo.FunctionStatusText005.TextValue)
	deviceStatus.WorkingMode = aq.translateCodeToString(r.StatusDataInfo.FunctionStatusText007.TextValue)
	deviceStatus.WaterInlet = r.StatusDataInfo.FunctionStatusText009.Value
	deviceStatus.WaterOutlet = r.StatusDataInfo.FunctionStatusText011.Value
	deviceStatus.Zone1ActualTemperature = r.StatusDataInfo.FunctionStatusText013.Value
	deviceStatus.Zone1SetpointTemperature = r.StatusDataInfo.FunctionStatusText015.Value
	deviceStatus.Zone1WaterTemperature = r.StatusDataInfo.FunctionStatusText017.Value
	deviceStatus.Zone2ActualTemperature = r.StatusDataInfo.FunctionStatusText019.Value
	deviceStatus.Zone2SetpointTemperature = r.StatusDataInfo.FunctionStatusText021.Value
	deviceStatus.Zone2WaterTemperature = r.StatusDataInfo.FunctionStatusText023.Value
	deviceStatus.DailyWaterTankActualTemperature = r.StatusDataInfo.FunctionStatusText025.Value
	deviceStatus.DailyWaterTankSetpointTemperature = r.StatusDataInfo.FunctionStatusText027.Value
	deviceStatus.BufferTankTemperature = r.StatusDataInfo.FunctionStatusText029.Value
	deviceStatus.OutdoorTemperature = r.StatusDataInfo.FunctionStatusText031.Value
	deviceStatus.CompressorStatus = "TODO__GDZIES MUSI BYC__/33 "
	deviceStatus.WaterFlow = r.StatusDataInfo.FunctionStatusText035.Value
	deviceStatus.PumpSpeed = r.StatusDataInfo.FunctionStatusText037.Value
	deviceStatus.HeatDirection = aq.translateCodeToString(r.StatusDataInfo.FunctionStatusText039.TextValue)
	deviceStatus.RoomHeaterStatus = aq.translateCodeToString(r.StatusDataInfo.FunctionStatusText041.TextValue)
	deviceStatus.DailyWaterHeaterStatus = aq.translateCodeToString(r.StatusDataInfo.FunctionStatusText043.TextValue)
	deviceStatus.DefrostStatus = aq.translateCodeToString(r.StatusDataInfo.FunctionStatusText045.TextValue)
	deviceStatus.SolarStatus = r.StatusDataInfo.FunctionStatusText047.Value
	deviceStatus.SolarTemperature = r.StatusDataInfo.FunctionStatusText049.Value
	deviceStatus.BiMode = r.StatusDataInfo.FunctionStatusText051.Value
	deviceStatus.ErrorStatus = r.StatusDataInfo.FunctionStatusText053.Value
	deviceStatus.CompressorFrequency = r.StatusDataInfo.FunctionStatusText056.Value
	deviceStatus.Runtime = r.StatusDataInfo.FunctionStatusText058.Value
	deviceStatus.RunCount = r.StatusDataInfo.FunctionStatusText060.Value
	deviceStatus.RoomHeaterPerformance = r.StatusDataInfo.FunctionStatusText063.Value
	deviceStatus.RoomHeaterRunTime = r.StatusDataInfo.FunctionStatusText065.Value
	deviceStatus.DailyWaterHeaterRunTime = r.StatusDataInfo.FunctionStatusText068.Value

	return deviceStatus, err
}

func (aq *aquarea) translateCodeToString(source string) string {
	if trans, found := aq.dictionaryWebUI[source]; found {
		return trans
	}
	return source
}

func (aq *aquarea) getDictionary(user endUserJSON) error {
	_, err := aq.getEndUserShiesuahruefutohkun(user)
	if err != nil {
		return err
	}
	body, err := aq.httpPost(aq.AquareaServiceCloudURL+"installer/functionStatus", nil)
	if err != nil {
		return err
	}
	return aq.extractDictionary(body)
}

func (aq *aquarea) getShiesuahruefutohkun(url string) (string, error) {
	body, err := aq.httpGet(url)
	if err != nil {
		return "", err
	}
	return aq.extractShiesuahruefutohkun(body)
}

func (aq *aquarea) getEndUserShiesuahruefutohkun(user endUserJSON) (string, error) {
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

	var loginStruct getLoginJSON
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
	var endUsersList endUsersListJSON
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

func (aq *aquarea) getDeviceStatus(user endUserJSON) (aquareaStatusResponseJSON, error) {

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

func (aq aquarea) getDeviceLogInformation(eu endUserJSON) (aquareaLog, error) {
	var aqLog aquareaLog
	var respo []string
	var AQLogData aquareaLogDataJSON
	shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(eu)

	sec := time.Now().Unix() // number of seconds since January 1, 1970 UTC
	lsec := sec - aq.logSecOffset
	ValueList := "{\"logItems\":[0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70]}"
	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/data/log", url.Values{
		"var.deviceId":        {eu.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
		"var.target":          {"0"},
		"var.startDate":       {fmt.Sprintf("%d000", lsec)},
		"var.logItems":        {ValueList},
	})
	if err != nil {
		return aqLog, err

	}
	err = json.Unmarshal(b, &AQLogData)

	var anything map[int64][]string
	err = json.Unmarshal([]byte(AQLogData.LogData), &anything)

	if len(anything) < 1 {
		return aqLog, nil

	}
	keys := make([]int64, 0, len(anything))
	for k := range anything {
		keys = append(keys, k)
	}
	//sort.Ints(keys)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	lastkey := len(keys) - 1

	respo = anything[keys[lastkey]]

	if err != nil {
		return aqLog, err

	}
	aqLog.logData = respo
	aqLog.timestamp = keys[lastkey]

	return aqLog, nil
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

func (aq aquarea) makeChangeHeatingTemperatureJSON(eui string, zoneid int, setpoint int) {
	eu := aq.usersMap[eui]

	var SetParam setParamJSON
	var ZS zoneStatusJSON
	ZS.HeatSet = setpoint
	ZS.ZoneID = zoneid
	ZST := []zoneStatusJSON{ZS}
	var ZSS spStatusJSON
	ZSS.DeviceGUID = eu.DeviceID
	ZSS.ZoneStatus = ZST
	SPS := []spStatusJSON{ZSS}
	SetParam.Status = SPS

	PAYLOAD, err := json.Marshal(SetParam)
	if err != nil {
		return
	}
	aq.setUserOption(eui, string(PAYLOAD))
}

// funkcja tylko do testow writow
func (aq aquarea) setUserOption(eui string, payload string) error {
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
