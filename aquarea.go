package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

type aquarea struct {
	config              configType
	client              http.Client
	dataChannel         chan extractedData
	logChannel          chan aquareaLog
	poolInterval        time.Duration
	shiesuahruefutohkun string
	lastChecksum        [16]byte
	logts               int64
}

type aquareaLog struct {
	LD []string
	TS int64
}

func (aq *aquarea) getAQData() bool {

	err := aq.getFirstShiesuahruefutohkun()
	if err != nil {
		log.Println(err)
		return false
	}

	err = aq.getLogin()
	if err != nil {
		log.Println(err)
		return false
	}

	EU, aqdict, err := aq.getInstallerHome()
	if err != nil {
		log.Println(err)
		return false
	}
	for {
		if err == nil {
			for _, SelectedEndUser := range EU {
				U, e := aq.parseAQData(SelectedEndUser, aq.client, aqdict)

				curLOGTS, LOGDATA, e := aq.getDeviceLogInformation(SelectedEndUser)
				if curLOGTS != aq.logts {
					aq.logChannel <- aquareaLog{LOGDATA, curLOGTS}
					aq.logts = curLOGTS
				}

				if e != nil {
					log.Println(e)
					return false
				}
				log.Printf("%s - ", U)
				md5 := md5.Sum([]byte(fmt.Sprintf("%s", U)))
				log.Printf("%x\n", md5)

				aqDevices[SelectedEndUser.Gwid] = SelectedEndUser

				//		go RandomSetTemp(client, SelectedEndUser)
				if md5 != aq.lastChecksum {
					aq.dataChannel <- U
					aq.lastChecksum = md5
				}
			}
		} else {
			log.Println(err)
		}
		time.Sleep(aq.poolInterval)
	}
}

func (aq aquarea) parseAQData(SelectedEndUser enduser, client http.Client, aqdict map[string]string) (extractedData, error) {
	var ED extractedData
	err := aq.getUIShiesuahruefutohkun(SelectedEndUser)
	r, err := aq.getDeviceInformation(SelectedEndUser)
	ED.EnduserID = SelectedEndUser.Gwid
	ED.RunningStatus = translateCodeToString(client, r.StatusDataInfo.FunctionStatusText005.TextValue)
	ED.WorkingMode = translateCodeToString(client, r.StatusDataInfo.FunctionStatusText007.TextValue)
	ED.WaterInlet = r.StatusDataInfo.FunctionStatusText009.Value
	ED.WaterOutlet = r.StatusDataInfo.FunctionStatusText011.Value
	ED.Zone1ActualTemperature = r.StatusDataInfo.FunctionStatusText013.Value
	ED.Zone1SetpointTemperature = r.StatusDataInfo.FunctionStatusText015.Value
	ED.Zone1WaterTemperature = r.StatusDataInfo.FunctionStatusText017.Value
	ED.Zone2ActualTemperature = r.StatusDataInfo.FunctionStatusText019.Value
	ED.Zone2SetpointTemperature = r.StatusDataInfo.FunctionStatusText021.Value
	ED.Zone2WaterTemperature = r.StatusDataInfo.FunctionStatusText023.Value
	ED.DailyWaterTankActualTemperature = r.StatusDataInfo.FunctionStatusText025.Value
	ED.DailyWaterTankSetpointTemperature = r.StatusDataInfo.FunctionStatusText027.Value
	ED.BufferTankTemperature = r.StatusDataInfo.FunctionStatusText029.Value
	ED.OutdoorTemperature = r.StatusDataInfo.FunctionStatusText031.Value
	ED.CompressorStatus = "TODO__GDZIES MUSI BYC__/33 "
	ED.WaterFlow = r.StatusDataInfo.FunctionStatusText035.Value
	ED.PumpSpeed = r.StatusDataInfo.FunctionStatusText037.Value
	ED.HeatDirection = translateCodeToString(client, r.StatusDataInfo.FunctionStatusText039.TextValue)
	ED.RoomHeaterStatus = translateCodeToString(client, r.StatusDataInfo.FunctionStatusText041.TextValue)
	ED.DailyWaterHeaterStatus = translateCodeToString(client, r.StatusDataInfo.FunctionStatusText043.TextValue)
	ED.DefrostStatus = translateCodeToString(client, r.StatusDataInfo.FunctionStatusText045.TextValue)
	ED.SolarStatus = r.StatusDataInfo.FunctionStatusText047.Value
	ED.SolarTemperature = r.StatusDataInfo.FunctionStatusText049.Value
	ED.BiMode = r.StatusDataInfo.FunctionStatusText051.Value
	ED.ErrorStatus = r.StatusDataInfo.FunctionStatusText053.Value
	ED.CompressorFrequency = r.StatusDataInfo.FunctionStatusText056.Value
	ED.Runtime = r.StatusDataInfo.FunctionStatusText058.Value
	ED.RunCount = r.StatusDataInfo.FunctionStatusText060.Value
	ED.RoomHeaterPerformance = r.StatusDataInfo.FunctionStatusText063.Value
	ED.RoomHeaterRunTime = r.StatusDataInfo.FunctionStatusText065.Value
	ED.DailyWaterHeaterRunTime = r.StatusDataInfo.FunctionStatusText068.Value
	if ED.RunCount == "-" {
		err = errors.New("Dane Wygladaja na BEZ TRESCI")
	}
	//fmt.Println("\n CODE: ", r.ErrorCode, "\n")
	//fmt.Println("\n BODY: ", r, "\n")

	return ED, err
}
func translateCodeToString(client http.Client, source string) string {
	// todo switch to download it everytime from aquarea
	aqdict := "{\"2006-01C0\":\"set\",\"2006-09C0\":\"Off\",\"2000-0045\":\"Unknown\",\"2006-0D00\":\"Room heater\",\"2000-0321\":\"IDU\",\"2000-0c09\":\"French\",\"2000-0c08\":\"Finnish\",\"2000-0c07\":\"Estonian\",\"2000-0c06\":\"English\",\"2000-0041\":\"Now processing…\",\"2000-0042\":\"Now processing…\",\"2006-09B0\":\"On\",\"2000-0c01\":\"Bulgarian\",\"2999-0094\":\"Terms of use\",\"2000-0c05\":\"Deutsch\",\"2006-0640\":\"Room heater\",\"2000-0c04\":\"Danish\",\"2999-0098\":\"Privacy Notice\",\"2000-0c03\":\"Czech\",\"2000-0c02\":\"Croatian\",\"2006-0120\":\"Mode\",\"2006-0910\":\"On\",\"2000-0311\":\"ID\",\"2006-0E10\":\"Operating time\",\"2006-01B0\":\"DHW tank\",\"2000-0391\":\"Monitoring + control\",\"2006-0190\":\"set\",\"2006-09A0\":\"Off\",\"2999-009b\":\"Cookie Policy\",\"2000-0031\":\"Set\",\"2006-0630\":\"3-way valve\",\"2006-0110\":\"Operation\",\"2006-0990\":\"On\",\"2006-0900\":\"Off\",\"2006-0348\":\"Auto (Cool) + Tank\",\"2999-00e0\":\"Logout\",\"2006-0E00\":\"Tank heater\",\"2000-0100\":\"Log out?\",\"2000-0221\":\"Status\",\"2006-0A00\":\"On\",\"2006-01A0\":\"water\",\"2000-0b09\":\"Spain\",\"2000-0b08\":\"Estonia\",\"2999-0038\":\"Registration\",\"2000-0060\":\"No user\",\"2000-0b07\":\"Denmark\",\"2000-0065\":\"AQUAREA Smart Cloud\",\"2000-0341\":\"Approved Full access Until\",\"2006-0180\":\"Zone2 temp.\",\"2000-0b02\":\"Belgium\",\"2000-0b01\":\"Austria\",\"2006-0620\":\"Pump speed\",\"2999-0030\":\"Customer\",\"2000-0b06\":\"Germany\",\"2006-0343\":\"Auto (Heat) + Tank\",\"2000-0b05\":\"Czech Republic\",\"2006-0100\":\"System status\",\"2006-0980\":\"Off\",\"2000-0b04\":\"Switzerland\",\"2999-0034\":\"List\",\"2000-0b03\":\"Bulgaria\",\"2000-0c0f\":\"Dutch\",\"2006-0339\":\"Heat + Tank\",\"2000-0c0a\":\"Hungarian\",\"2006-033E\":\"Cool + Tank\",\"2000-0331\":\"ODU\",\"2000-0211\":\"User information\",\"2000-0c0e\":\"Lithuanian\",\"2999-003b\":\"Delete\",\"2000-0c0d\":\"Latvian\",\"2000-0c0c\":\"Italian\",\"2006-0C30\":\"Number of operations\",\"2000-0c0b\":\"Irish\",\"2000-0050\":\"AQUAREA Service Cloud\",\"2000-0c19\":\"Greek\",\"2006-0690\":\"Bivalent\",\"2000-0c18\":\"Turkish\",\"2000-0c17\":\"Swedish\",\"2006-0170\":\"water\",\"2000-0055\":\"AQUAREA Service Cloud\",\"2006-032F\":\"Auto (Heat)\",\"2000-0c12\":\"Portuguese\",\"2000-0c11\":\"Polish\",\"2006-0610\":\"Water flow\",\"2000-0c10\":\"Norwegian\",\"2006-0334\":\"Auto (Cool)\",\"2000-0c16\":\"Spanish\",\"2006-0970\":\"On\",\"2000-0c15\":\"Slovenian\",\"2000-0c14\":\"Slovak\",\"2000-0c13\":\"Romanian\",\"2000-0125\":\"The device has been deleted.\",\"2000-0b1b\":\"Turkey\",\"2006-06A0\":\"Error\",\"2000-0001\":\"Cancel\",\"2000-0b1a\":\"Finland\",\"2006-032A\":\"Cool\",\"2006-0C20\":\"Operating time\",\"2000-0401\":\"Full access\",\"2000-0005\":\"OK\",\"2006-0680\":\"Solar temp.\",\"2006-0160\":\"set\",\"2000-0120\":\"Delete this device from this service?\",\"2000-0241\":\"Data log\",\"2000-0361\":\"Access rights\",\"2006-0600\":\"Thermo\",\"2006-0325\":\"Heat\",\"2000-0a01\":\"Off\",\"2006-0960\":\"Off\",\"2006-0320\":\"Tank\",\"2000-0a05\":\"On\",\"2000-0b0b\":\"United Kingdom\",\"2000-0b0a\":\"France\",\"2006-09F0\":\"Off\",\"2000-0b0f\":\"Italy\",\"2006-0C10\":\"Compressor frequency\",\"2000-0b0e\":\"Ireland\",\"2000-0b0d\":\"Hungary\",\"2000-0b0c\":\"Croatia\",\"2000-0070\":\"AQUAREA Smart Cloud\",\"2000-0b19\":\"Slovakia\",\"2006-0150\":\"Zone1 temp.\",\"2000-0b18\":\"Slovenia\",\"2000-0351\":\"Waiting for approval\",\"2000-0110\":\"Return to login page?\",\"2000-0231\":\"Statistics\",\"2999-0060\":\"Company\",\"2000-0b13\":\"Norway\",\"2000-0b12\":\"Netherlands\",\"2000-0b11\":\"Latvia\",\"2006-0950\":\"Tank\",\"2000-0b10\":\"Lithuania\",\"2000-0b17\":\"Sweden\",\"2006-0310\":\"On\",\"2000-0b16\":\"Romania\",\"2000-0b15\":\"Portugal\",\"2000-0b14\":\"Poland\",\"2006-0670\":\"Solar\",\"2006-01E0\":\"Outdoor temp.\",\"2000-0a1c\":\"Dec\",\"2000-0a1b\":\"Nov\",\"2000-0a1a\":\"Oct\",\"2006-0C00\":\"Compressor\",\"2006-0D20\":\"Operating time\",\"2000-0381\":\"Monitoring only\",\"2006-0140\":\"Outlet water\",\"2000-0021\":\"Send\",\"2006-0940\":\"Room\",\"2006-0300\":\"Off\",\"2000-012f\":\"Device ID\",\"2006-0660\":\"Defrost\",\"2006-01D0\":\"Buffer tank\",\"2999-0090\":\"Agreement\",\"2000-012a\":\"Customer name\",\"2000-0015\":\"Agree\",\"2006-09D0\":\"On\",\"2006-0D10\":\"Heater capacity\",\"2999-00b0\":\"Help\",\"2006-0130\":\"Inlet water\",\"2000-0a19\":\"Sep\",\"2000-0011\":\"Disagree\",\"2000-0371\":\"On request\",\"2000-0251\":\"Setting\",\"2000-0a14\":\"Apr\",\"2000-0a13\":\"Mar\",\"2000-0a12\":\"Feb\",\"2000-0a11\":\"Jan\",\"2000-0a18\":\"Aug\",\"2000-0a17\":\"Jul\",\"2006-0650\":\"Tank heater\",\"2999-0000\":\"Menu\",\"2000-0a16\":\"Jun\",\"2000-0a15\":\"May\"}"
	var m map[string]string

	//	client := http.Client{
	//		Jar: cookieJar,
	//	}
	//	resp, err := client.Get(config.AquareaServiceCloudURL + "installer/functionStatus")
	//	if err != nil {
	//		return source
	//	}
	//	defer resp.Body.Close()
	//	body, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	//	re33, err := regexp.Compile(`const jsonMessage = eval\('\((.+)\)'`)
	//	ss33 := re33.FindAllStringSubmatch(string(body), -1)
	//	if len(ss33) > 0 {

	//result := strings.Replace(ss33[0], "\\", "", -1)
	//		fmt.Println(ss33[0][0])
	//

	//	}
	//	fmt.Println(m)

	err := json.Unmarshal([]byte(aqdict), &m)
	if err != nil {
		fmt.Println("BLAD:", err, aqdict)
		return source
	}
	if _, found := m[source]; !found {
		return source
	}
	return m[source]
}

func (aq *aquarea) getFirstShiesuahruefutohkun() error {
	req, err := http.NewRequest("GET", aq.config.AquareaServiceCloudURL, nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:74.0) Gecko/20100101 Firefox/74.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := aq.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	re, err := regexp.Compile(`const shiesuahruefutohkun = '(.+)'`)
	ss := re.FindStringSubmatch(string(body))

	if len(ss) > 0 {
		aq.shiesuahruefutohkun = ss[1]
	}
	return err
}

func (aq *aquarea) getUIShiesuahruefutohkun(eu enduser) error {

	LoginURL := aq.config.AquareaServiceCloudURL + "/installer/functionUserInformation"
	uv := url.Values{
		"var.functionSelectedGwUid": {eu.GwUID},
	}
	resp, err := postREQ(LoginURL, aq.client, uv)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	re, err := regexp.Compile(`const shiesuahruefutohkun = '(.+)'`)
	ss := re.FindStringSubmatch(string(body))
	if len(ss) > 0 {
		aq.shiesuahruefutohkun = ss[1]
	}
	return err

}

func (aq aquarea) getLogin() error {

	var Response getLoginStruct
	LoginURL := aq.config.AquareaServiceCloudURL + "installer/api/auth/login"
	data := []byte(aq.config.AquareaServiceCloudLogin + aq.config.AquareaServiceCloudPassword)

	uv := url.Values{
		"var.loginId":         {aq.config.AquareaServiceCloudLogin},
		"var.password":        {fmt.Sprintf("%x", md5.Sum(data))},
		"var.inputOmit":       {"false"},
		"shiesuahruefutohkun": {aq.shiesuahruefutohkun},
	}

	resp, err := postREQ(LoginURL, aq.client, uv)
	if err != nil {
		return err

	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(b))
	defer resp.Body.Close()

	err = json.Unmarshal(b, &Response)
	fmt.Println(err, "tooooo", string(b))

	if Response.ErrorCode != 0 {
		err = fmt.Errorf("%d", Response.ErrorCode)
	}

	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (aq aquarea) getInstallerHome() ([]enduser, map[string]string, error) {

	var EndUsersList endUsersList
	var EndUsers []enduser
	var err error
	var m map[string]string

	var Shiesuahruefutohkun string
	req, err := http.NewRequest("GET", aq.config.AquareaServiceCloudURL+"installer/home", nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:74.0) Gecko/20100101 Firefox/74.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "deflate, br")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := aq.client.Do(req)
	if err != nil {
		return EndUsers, m, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	re, err := regexp.Compile(`const shiesuahruefutohkun = '(.+)'`)
	ss := re.FindStringSubmatch(string(body))

	re33, err := regexp.Compile(`const jsonMessage = eval\('\((.+)\)'`)
	ss33 := re33.FindStringSubmatch(string(body))
	if len(ss33) > 0 {
		result := strings.Replace(ss33[1], "\\", "", -1)
		err = json.Unmarshal([]byte(result), &m)
		fmt.Println(err, "tutaj", result)
	}

	if len(ss) > 0 {
		Shiesuahruefutohkun = ss[1]
	} else {
		err = errors.New("BLAD")
	}
	resp.Body.Close()

	LoginURL := aq.config.AquareaServiceCloudURL + "/installer/api/endusers"

	req, err = http.NewRequest("POST", LoginURL, strings.NewReader(url.Values{
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
		"shiesuahruefutohkun": {Shiesuahruefutohkun},
	}.Encode()))
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:74.0) Gecko/20100101 Firefox/74.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "deflate, br")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = aq.client.Do(req)

	if err != nil {
		return EndUsers, m, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return EndUsers, m, err
	}
	err = json.Unmarshal(b, &EndUsersList)
	if err != nil {
		fmt.Println(err, string(b))
		return EndUsers, m, err
	}
	EndUsers = EndUsersList.Endusers

	resp.Body.Close()

	return EndUsers, m, err
}

func (aq aquarea) getDeviceInformation(eu enduser) (aquareaStatusResponse, error) {

	var AquareaStatusResponse aquareaStatusResponse

	LoginURL := aq.config.AquareaServiceCloudURL + "/installer/api/function/status"
	uv := url.Values{
		"var.deviceId":        {eu.DeviceID},
		"shiesuahruefutohkun": {aq.shiesuahruefutohkun},
	}
	resp, err := postREQ(LoginURL, aq.client, uv)
	if err != nil {
		return AquareaStatusResponse, err

	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return AquareaStatusResponse, err

	}
	SB := string(b)
	err = json.Unmarshal([]byte(SB), &AquareaStatusResponse)
	fmt.Println(err, SB)

	//fmt.Println(AquareaStatusResponse.StatusDataInfo.FunctionStatusText027.Value)
	resp.Body.Close()

	return AquareaStatusResponse, err
}

func (aq aquarea) getDeviceLogInformation(eu enduser) (int64, []string, error) {
	var respo []string
	var AQLogData aqLogData
	sec := time.Now().Unix() // number of seconds since January 1, 1970 UTC
	lsec := sec - aq.config.LogSecOffset
	ValueList := "{\"logItems\":[0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70]}"
	LoginURL := aq.config.AquareaServiceCloudURL + "/installer/api/data/log"
	uv := url.Values{
		"var.deviceId":        {eu.DeviceID},
		"shiesuahruefutohkun": {aq.shiesuahruefutohkun},
		"var.target":          {"0"},
		"var.startDate":       {fmt.Sprintf("%d000", lsec)},
		"var.logItems":        {ValueList},
	}
	resp, err := postREQ(LoginURL, aq.client, uv)
	if err != nil {
		return sec, respo, err

	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return sec, respo, err

	}
	err = json.Unmarshal(b, &AQLogData)
	fmt.Println(err, b)

	var anything map[int64][]string
	err = json.Unmarshal([]byte(AQLogData.LogData), &anything)
	fmt.Println(err, AQLogData.LogData)

	if len(anything) < 1 {
		return sec, respo, nil

	}
	keys := make([]int64, 0, len(anything))
	for k := range anything {
		keys = append(keys, k)
	}
	//sort.Ints(keys)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	lastkey := len(keys) - 1

	fmt.Println(keys)
	fmt.Println(keys[lastkey])

	respo = anything[keys[lastkey]]
	resp.Body.Close()

	if err != nil {
		return sec, respo, err

	}
	return keys[lastkey], respo, nil
}

func postREQ(LoginURL string, client http.Client, uv url.Values) (*http.Response, error) {
	req, err := http.NewRequest("POST", LoginURL, strings.NewReader(uv.Encode()))
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:74.0) Gecko/20100101 Firefox/74.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "deflate, br")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	return resp, err
}

func makeChangeHeatingTemperatureJSON(eui string, zoneid int, setpoint int) string {
	eu := aqDevices[eui]

	var SetParam setParam
	var ZS zoneStatus
	ZS.HeatSet = setpoint
	ZS.ZoneID = zoneid
	ZST := []zoneStatus{ZS}
	var ZSS spStatus
	ZSS.DeviceGUID = eu.DeviceID
	ZSS.ZoneStatus = ZST
	SPS := []spStatus{ZSS}
	SetParam.Status = SPS

	PAYLOAD, err := json.Marshal(SetParam)
	if err != nil {
		return "ERR"
	}
	return string(PAYLOAD)
}

// funkcja tylko do testow writow
func (aq aquarea) setUserOption(eui string, payload string) error {
	eu := aqDevices[eui]
	var AQCSR aquareaServiceCloudSSOReponse

	_, err := aq.client.Get(aq.config.AquareaServiceCloudURL + "enduser/confirmStep1Policy")
	CreateSSOUrl := aq.config.AquareaServiceCloudURL + "/enduser/api/request/create/sso"
	uv := url.Values{
		"var.gwUid":           {eu.GwUID},
		"shiesuahruefutohkun": {aq.shiesuahruefutohkun},
	}
	resp, err := postREQ(CreateSSOUrl, aq.client, uv)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &AQCSR)
	log.Println(err, body)

	leadInstallerStep1url := aq.config.AquareaSmartCloudURL + "/remote/leadInstallerStep1"
	uv = url.Values{
		"var.keyCode": {AQCSR.SsoKey},
	}
	_, err = postREQ(leadInstallerStep1url, aq.client, uv)
	ClaimSSOurl := aq.config.AquareaSmartCloudURL + "/remote/v1/api/auth/sso"
	uv = url.Values{
		"var.ssoKey": {AQCSR.SsoKey},
	}
	_, err = postREQ(ClaimSSOurl, aq.client, uv)
	a2wStatusDisplayurl := aq.config.AquareaSmartCloudURL + "/remote/a2wStatusDisplay"
	uv = url.Values{}
	_, err = postREQ(a2wStatusDisplayurl, aq.client, uv)
	_, err = aq.client.Get(aq.config.AquareaSmartCloudURL + "/service-worker.js")
	url := aq.config.AquareaSmartCloudURL + "/remote/v1/api/devices/" + eu.DeviceID

	//var jsonStr = []byte(`{"status":[{"deviceGuid":"008007B767718332001434545313831373030634345373130434345373138313931304300000000","zoneStatus":[{"zoneId":1,"heatSet":25}]}]}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Referer", aq.config.AquareaSmartCloudURL+"/remote/a2wControl")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,pl;q=0.8,zh;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", aq.config.AquareaSmartCloudURL)
	req.Header.Set("Content-Type", "application/json")

	resp, err = aq.client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(http.StatusText(resp.StatusCode))
	}
	return nil
}
