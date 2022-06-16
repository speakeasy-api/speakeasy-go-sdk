package speakeasy

var Config internalConfiguration

const defaultServerURL = "localhost:3000" // TODO: Inject appropriate Speakeasy Registry API endpoint either through env vars

// Configuration sets up and customizes communication with the Speakeasy Registry API
type Configuration struct {
	APIKey      string
	WorkspaceId string
	KeysToMask  []string
	ServerURL   string
}

// internalConfiguration is used for communication with Speakeasy Registry API
type internalConfiguration struct {
	Configuration
	KeysMap      map[string]interface{}
	serverInfo   ServerInfo
	languageInfo LanguageInfo
}

func Configure(config Configuration) {
	if config.ServerURL != "" {
		Config.ServerURL = config.ServerURL
	} else {
		Config.ServerURL = defaultServerURL
	}
	if config.APIKey != "" {
		Config.APIKey = config.APIKey
	}
	if config.WorkspaceId != "" {
		Config.WorkspaceId = config.WorkspaceId
	}
	if len(config.KeysToMask) > 0 {
		Config.KeysToMask = config.KeysToMask

		// transform the string slice to a map for faster retrieval
		Config.KeysMap = make(map[string]interface{})
		for _, v := range config.KeysToMask {
			Config.KeysMap[v] = nil
		}
	}

	Config.serverInfo = getServerInfo()
	Config.languageInfo = getLanguageInfo()
}
