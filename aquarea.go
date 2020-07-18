package main

import (
	"context"
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
	"sync"
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
	Name          string            `json:"name"`
	Kind          string            `json:"kind"`
	Values        map[string]string `json:"values"`
	reverseValues map[string]string
}

type aquareaLogItem struct {
	Name   string
	Unit   string
	Values map[string]string
}

type aquarea struct {
	AquareaServiceCloudURL      string
	AquareaServiceCloudLogin    string
	AquareaServiceCloudPassword string
	logSecOffset                int64
	dataChannel                 chan map[string]string
	statusChannel               chan bool

	httpClient             http.Client
	dictionaryWebUI        map[string]string                      // xxxx-yyyy codes translation to messages
	reverseDictionaryWebUI map[string]string                      // message to xxxx-yyyy code translation
	usersMap               map[string]aquareaEndUserJSON          // list of users (devices) linked to an account
	translation            map[string]*aquareaFunctionDescription // function name meaning
	reverseTranslation     map[string]string                      // map of friendly names to Aquarea meaningless ones
	logItems               []aquareaLogItem                       // table with names of log items (statistics view)
	aquareaSettings        aquareaFunctionSettingGetJSON          // needs be cached, contains info relevant for changing settings
}

func aquareaHandler(ctx context.Context, wg *sync.WaitGroup, config configType, dataChannel chan map[string]string, commandChannel chan aquareaCommand, statusChannel chan bool) {
	defer wg.Done()
	log.Println("Starting Aquarea Service Cloud handler")
	var aquareaInstance aquarea
	aquareaInstance.AquareaServiceCloudURL = config.AquareaServiceCloudURL
	aquareaInstance.AquareaServiceCloudLogin = config.AquareaServiceCloudLogin
	aquareaInstance.AquareaServiceCloudPassword = config.AquareaServiceCloudPassword
	aquareaInstance.logSecOffset = config.LogSecOffset
	aquareaInstance.dataChannel = dataChannel
	aquareaInstance.statusChannel = statusChannel
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
	}
	log.Println("Logged in to Aquarea Service Cloud")

	ticker := time.NewTicker(poolInterval)
	for {
		select {
		case <-ticker.C:
			aquareaInstance.feedDataFromAquarea()
		case command := <-commandChannel:
			aquareaInstance.sendSetting(command)
		case <-ctx.Done():
			return
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

	// add reverse Value translation
	for kDescr, descr := range aq.translation {
		aq.translation[kDescr].reverseValues = make(map[string]string)
		for k, v := range descr.Values {
			descr.reverseValues[v] = k
		}
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
		shiesuahruefutohkun, err := aq.getEndUserShiesuahruefutohkun(user)
		if err != nil {
			aq.statusChannel <- false
			log.Println(err)
			log.Println("Will attempt to log in again")
			// try to log in again
			aq.aquareaSetup()
			continue
		}

		settings, err := aq.getDeviceSettings(user, shiesuahruefutohkun)
		if err != nil {
			log.Println(err)
		} else {
			aq.dataChannel <- settings
		}

		// Send device status
		deviceStatus, err := aq.parseDeviceStatus(user, shiesuahruefutohkun)
		if err != nil {
			log.Println(err)
		} else {
			aq.dataChannel <- deviceStatus
		}

		// Send device logs
		logData, err := aq.getDeviceLogInformation(user, shiesuahruefutohkun)
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
