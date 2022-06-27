package speakeasy

import (
	"runtime"
)

type MetaData struct {
	ApiKey  string   `json:"api_key"`
	Version float32  `json:"version"`
	Sdk     string   `json:"sdk"`
	Data    DataInfo `json:"data"`
}

type ApiData struct {
	ApiKey      string        `json:"api_key"`
	ApiServerId string        `json:"api_server_id"`
	Handlers    []HandlerInfo `json:"handlers"`
}

type HandlerInfo struct {
	// TODO: This should be api_id instead of path once we register Apis from speakeasy.Configure
	Path     string   `json:"path"`
	ApiStats ApiStats `json:"api_info"`
}

type ApiStats struct {
	NumCalls           int `json:"number_of_calls"`
	NumErrors          int `json:"number_of_errors"`
	NumUniqueCustomers int `json:"number_of_unique_customers"`
}

type DataInfo struct {
	Server   ServerInfo   `json:"server"`
	Language LanguageInfo `json:"language"`
	Request  RequestInfo  `json:"request"`
	Response ResponseInfo `json:"response"`
}

type ServerInfo struct {
	Ip        string `json:"ip"`
	Timezone  string `json:"timezone"`
	Software  string `json:"software"`
	Signature string `json:"signature"`
	Protocol  string `json:"protocol"`
	Os        OsInfo `json:"os"`
}

type OsInfo struct {
	Name         string `json:"name"`
	Release      string `json:"release"`
	Architecture string `json:"architecture"`
}

type LanguageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Get information about the server environment
func getServerInfo() ServerInfo {
	return ServerInfo{
		Ip:        "",
		Timezone:  "UTC",
		Software:  "",
		Signature: "",
		Protocol:  "",
		Os:        getOsInfo(),
	}
}

// Get information about the programming language
func getLanguageInfo() LanguageInfo {
	return LanguageInfo{
		Name:    "go",
		Version: runtime.Version(),
	}
}

// Get information about the operating system that is running on the server
func getOsInfo() OsInfo {
	return OsInfo{
		Name:         runtime.GOOS,
		Release:      "",
		Architecture: runtime.GOARCH,
	}
}
