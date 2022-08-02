package speakeasy

var ExportServerURL = serverURL

const ExportMaxIDSize = maxIDSize

func (s *speakeasy) ExportGetSpeakeasyConfig() Config {
	return s.config
}

func (s *speakeasy) ExportGetSpeakeasyServerURL() string {
	return s.serverURL
}

func (s *speakeasy) ExportGetSpeakeasyServerSecure() bool {
	return s.secure
}
