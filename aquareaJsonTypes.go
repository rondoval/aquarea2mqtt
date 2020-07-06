package main

type aquareaEndUsersListJSON struct {
	ZoomMap            int                  `json:"zoomMap"`
	ErrorCode          int                  `json:"errorCode"`
	Endusers           []aquareaEndUserJSON `json:"endusers"`
	LongitudeCenterMap string               `json:"longitudeCenterMap"`
	Size               int                  `json:"size"`
	LatitudeCenterMap  string               `json:"latitudeCenterMap"`
}
type aquareaEndUserJSON struct {
	Address    string      `json:"address"`
	CompanyID  string      `json:"companyId"`
	Connection string      `json:"connection"`
	DeviceID   string      `json:"deviceId"`
	EnduserID  string      `json:"enduserId"`
	ErrorCode  interface{} `json:"errorCode"`
	ErrorName  string      `json:"errorName"`
	GwUID      string      `json:"gwUid"`
	Gwid       string      `json:"gwid"`
	Idu        string      `json:"idu"`
	Latitude   string      `json:"latitude"`
	Longitude  string      `json:"longitude"`
	Name       string      `json:"name"`
	Odu        string      `json:"odu"`
	Power      string      `json:"power"`
}

type aquareaStatusResponseJSON struct {
	ErrorCode      int `json:"errorCode"`
	StatusDataInfo map[string]struct {
		Value     string `json:"value"`
		TextValue string `json:"textValue"`
		Type      string `json:"type"`
	} `json:"statusDataInfo"`
	StatusBackgroundDataInfo map[string]struct {
		Value string `json:"value"`
	} `json:"statusBackgroundDataInfo"`
}

type logResponseJSON struct {
	ErrorCode int `json:"errorCode"`
	Message   []struct {
		ErrorMessage string `json:"errorMessage"`
		ErrorCode    string `json:"errorCode"`
	} `json:"message"`
}

type aquareaLogDataJSON struct {
	ErrorHistory []struct {
		ErrorCode string `json:"errorCode"`
		ErrorDate int64  `json:"errorDate"`
	} `json:"errorHistory"`
	LogData         string `json:"logData"`
	ErrorCode       int    `json:"errorCode"`
	RecordingStatus int    `json:"recordingStatus"`
	HistoryNo       string `json:"historyNo"`
}

// These below are for changing settings of the heat pump
type setParamJSON struct {
	Status []spStatusJSON `json:"status"`
}
type zoneStatusJSON struct {
	ZoneID  int `json:"zoneId"`
	HeatSet int `json:"heatSet"`
}
type spStatusJSON struct {
	DeviceGUID string           `json:"deviceGuid"`
	ZoneStatus []zoneStatusJSON `json:"zoneStatus"`
}

type aquareaServiceCloudSSOReponseJSON struct {
	SsoKey    string `json:"ssoKey"`
	ErrorCode int    `json:"errorCode"`
}

type getLoginJSON struct {
	AgreementStatus struct {
		Contract      bool `json:"contract"`
		CookiePolicy  bool `json:"cookiePolicy"`
		PrivacyPolicy bool `json:"privacyPolicy"`
	} `json:"agreementStatus"`
	ErrorCode int `json:"errorCode"`
}
