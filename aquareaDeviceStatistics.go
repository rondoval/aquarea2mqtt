package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Gets most recent data from the statistics page
func (aq *aquarea) getDeviceLogInformation(user aquareaEndUserJSON, shiesuahruefutohkun string) (map[string]string, error) {
	// Build list of all possible values to log
	var valueList strings.Builder
	valueList.WriteString("{\"logItems\":[")
	for i := range aq.logItems {
		valueList.WriteString(strconv.Itoa(i))
		valueList.WriteString(",")
	}
	valueList.WriteString("]}")

	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/data/log", url.Values{
		"var.deviceId":        {user.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
		"var.target":          {"0"},
		"var.startDate":       {fmt.Sprintf("%d000", time.Now().Unix()-aq.logSecOffset)},
		"var.logItems":        {valueList.String()},
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
		topic := fmt.Sprintf("aquarea/%s/log/", user.Gwid) + aq.logItems[i].Name

		if aq.logItems[i].Unit != "" {
			stats[topic+"/unit"] = aq.logItems[i].Unit // unit of the value, extracted from name
		}

		if x, ok := aq.logItems[i].Values[val]; ok {
			val = x
		}

		stats[topic] = val
	}
	stats[fmt.Sprintf("aquarea/%s/log/Timestamp", user.Gwid)] = strconv.FormatInt(lastKey, 10)
	stats[fmt.Sprintf("aquarea/%s/log/CurrentError", user.Gwid)] = strconv.Itoa(aquareaLogData.ErrorCode)
	return stats, nil
}
