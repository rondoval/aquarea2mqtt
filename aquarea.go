package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
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
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	logSecOffset                int64
	dataChannel                 chan map[string]string

	httpClient         http.Client
	dictionaryWebUI    map[string]string                     // xxxx-yyyy codes translation
	usersMap           map[string]aquareaEndUserJSON         // list of users (devices) linked to an account
	translation        map[string]aquareaFunctionDescription // function name meaning
	reverseTranslation map[string]string                     // map of friendly names to Aquarea meaningless ones
	logItems           []string                              // table with names of log items (statistics view)
	aquareaSettings    aquareaFunctionSettingGetJSON         // needs be cached, contains info relevant for changing settings
}

func aquareaHandler(config configType, dataChannel chan map[string]string, commandChannel chan aquareaCommand) {
	log.Println("Starting Aquarea Service Cloud handler")
	var aquareaInstance aquarea
	aquareaInstance.AquareaServiceCloudURL = config.AquareaServiceCloudURL
	aquareaInstance.AquareaServiceCloudLogin = config.AquareaServiceCloudLogin
	aquareaInstance.AquareaServiceCloudPassword = config.AquareaServiceCloudPassword
	aquareaInstance.logSecOffset = config.LogSecOffset
	aquareaInstance.dataChannel = dataChannel
	aquareaInstance.usersMap = make(map[string]aquareaEndUserJSON)

	aquareaInstance.loadTranslations(translationFile)

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

	log.Println("Attempting to log in to Aquarea Service Cloud")
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

func (aq *aquarea) loadTranslations(filename string) {
	// Load JSON with translations from Aquarea cryptic names
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &aq.translation)
	if err != nil {
		log.Fatal(err)
	}

	// create reverse map i.e. aquareaFunctionDescription.Name to key
	aq.reverseTranslation = make(map[string]string)
	for key, value := range aq.translation {
		if strings.Contains(key, "setting-user-select") {
			// can't do a reverse map of everything as there are duplicates
			// besides, we need it for settings only
			aq.reverseTranslation[value.Name] = key
		}
	}
}

func (aq *aquarea) feedDataFromAquarea() {
	for _, user := range aq.usersMap {
		// Get settings from the device
		settings, err := aq.receiveSettings(user)
		if err != nil {
			log.Println(err)
		} else {
			aq.dataChannel <- settings
		}

		// Send device status
		deviceStatus, err := aq.parseDeviceStatus(user)
		if err != nil {
			log.Println(err)
		} else {
			aq.dataChannel <- deviceStatus
		}

		// Send device logs
		logData, err := aq.getDeviceLogInformation(user)
		if err != nil {
			log.Println(err)
		} else {
			aq.dataChannel <- logData
		}
	}
}

func (aq *aquarea) getShiesuahruefutohkun(url string) (string, error) {
	body, err := aq.httpGet(url)
	if err != nil {
		return "", err
	}
	return aq.extractShiesuahruefutohkun(body)
}

//TODO try to minimize number of requests
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
	re := regexp.MustCompile(`const shiesuahruefutohkun = '(.+)'`)
	ss := re.FindStringSubmatch(string(body))
	if len(ss) > 0 {
		return ss[1], nil
	}
	return "", fmt.Errorf("Could not extract shiesuahruefutohkun")
}
