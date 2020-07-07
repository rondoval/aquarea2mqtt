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

type aquareaLoginJSON struct {
	AgreementStatus struct {
		Contract      bool `json:"contract"`
		CookiePolicy  bool `json:"cookiePolicy"`
		PrivacyPolicy bool `json:"privacyPolicy"`
	} `json:"agreementStatus"`
	ErrorCode int `json:"errorCode"`
}

// Settings using Service Cloud API
type aquareaFunctionSettingGetJSON struct {
	SettingDataInfo map[string]struct {
		Type          string            `json:"type"` // select (selectedValue), basic-text(textValue), placeholder-text (placeholder, params)
		SelectedValue string            `json:"selectedValue"`
		Placeholder   string            `json:"placeholder"`
		Params        map[string]string `json:"params"`
		TextValue     string            `json:"textValue"`
	} `json:"settingDataInfo"`
	SettingsBackgroundData map[string]struct {
		Value string `json:"value"`
	} `json:"settingBackgroundData"`
	ErrorCode int `json:"errorCode"`
}
