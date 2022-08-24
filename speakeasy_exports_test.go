package speakeasy

var ExportServerURL = serverURL

const ExportMaxIDSize = maxIDSize

func (s *Speakeasy) ExportGetSpeakeasyConfig() Config {
	return s.config
}

func (s *Speakeasy) ExportGetSpeakeasyServerURL() string {
	return s.grpcClient.serverURL
}

func (s *Speakeasy) ExportGetSpeakeasyServerSecure() bool {
	return s.grpcClient.secure
}
