package client

type Config struct {
	Endpoint     string
	NTPAddr      string
	CertPath     string
	ServerName   string
	Sync         bool
	SyncFix      int
	SyncInterval int
}

func (conf *Config) Check() error {
	return nil
}
