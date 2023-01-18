package utils

type ErrorMessage struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type ApiError struct {
	Message  string `json:"message"`
	Response string `json:"response"`
}
type ApiMessage struct {
	Message  string `json:"message"`
	Response string `json:"response"`
	Data     interface{} `json:"data"`
}


// Api Default Ok

func ApiDefaultMsgResponse(data interface{}, message string) *ApiMessage {
	return &ApiMessage{Response: "OK", Message: message, Data: data}
}

func ApiDefaultResponse(data interface{}) *ApiMessage {
	return &ApiMessage{Response: "OK", Data: data}
}

func ApiDefaultMsgOnly(message string) *ApiMessage {
	return &ApiMessage{Response: "OK", Message: message}
}