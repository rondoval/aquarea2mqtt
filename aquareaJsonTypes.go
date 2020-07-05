package main

type endUsersListJSON struct {
	ZoomMap            int           `json:"zoomMap"`
	ErrorCode          int           `json:"errorCode"`
	Endusers           []endUserJSON `json:"endusers"`
	LongitudeCenterMap string        `json:"longitudeCenterMap"`
	Size               int           `json:"size"`
	LatitudeCenterMap  string        `json:"latitudeCenterMap"`
}
type endUserJSON struct {
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
	StatusDataInfo struct {
		FunctionStatusText005 struct {
			TextValue string `json:"textValue"`
			Type      string `json:"type"`
		} `json:"function-status-text-005"`
		FunctionStatusText027 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-027"`
		FunctionStatusText049 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-049"`
		FunctionStatusText025 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-025"`
		FunctionStatusText047 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-047"`
		FunctionStatusText068 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-068"`
		FunctionStatusText009 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-009"`
		FunctionStatusText007 struct {
			TextValue string `json:"textValue"`
			Type      string `json:"type"`
		} `json:"function-status-text-007"`
		FunctionStatusText029 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-029"`
		FunctionStatusText041 struct {
			TextValue string `json:"textValue"`
			Type      string `json:"type"`
		} `json:"function-status-text-041"`
		FunctionStatusText063 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-063"`
		FunctionStatusText060 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-060"`
		FunctionStatusText023 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-023"`
		FunctionStatusText045 struct {
			TextValue string `json:"textValue"`
			Type      string `json:"type"`
		} `json:"function-status-text-045"`
		FunctionStatusText021 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-021"`
		FunctionStatusText043 struct {
			TextValue string `json:"textValue"`
			Type      string `json:"type"`
		} `json:"function-status-text-043"`
		FunctionStatusText065 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-065"`
		FunctionStatusText015 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-015"`
		FunctionStatusText037 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-037"`
		FunctionStatusText058 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-058"`
		FunctionStatusText013 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-013"`
		FunctionStatusText035 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-035"`
		FunctionStatusText019 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-019"`
		FunctionStatusText017 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-017"`
		FunctionStatusText039 struct {
			TextValue string `json:"textValue"`
			Type      string `json:"type"`
		} `json:"function-status-text-039"`
		FunctionStatusText051 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-051"`
		FunctionStatusText056 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-056"`
		FunctionStatusText011 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-011"`
		FunctionStatusText031 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-031"`
		FunctionStatusText053 struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"function-status-text-053"`
	} `json:"statusDataInfo"`
	StatusBackgroundDataInfo struct {
		ZeroXA0 struct {
			Value string `json:"value"`
		} `json:"0xA0"`
		ZeroX20 struct {
			Value string `json:"value"`
		} `json:"0x20"`
		ZeroXE1 struct {
			Value string `json:"value"`
		} `json:"0xE1"`
		ZeroXE0 struct {
			Value string `json:"value"`
		} `json:"0xE0"`
		ZeroXFA struct {
			Value string `json:"value"`
		} `json:"0xFA"`
		ZeroXF0 struct {
			Value string `json:"value"`
		} `json:"0xF0"`
		ZeroX80 struct {
			Value string `json:"value"`
		} `json:"0x80"`
		ZeroXF9 struct {
			Value string `json:"value"`
		} `json:"0xF9"`
		ZeroXC4 struct {
			Value string `json:"value"`
		} `json:"0xC4"`
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
