package speakeasy

var ExportServerURL = serverURL

func ExportGetSpeakeasyDefaultInstance() *speakeasy {
	return defaultInstance
}

func ExportResetSpeakeasyDefaultInstance() {
	defaultInstance = nil
}

func (s *speakeasy) ExportGetSpeakeasyConfig() Config {
	return s.config
}

func (s *speakeasy) ExportGetSpeakeasyServerURL() string {
	return s.serverURL.String()
}
