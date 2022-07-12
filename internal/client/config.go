package client

type Config struct {
	Endpoint   string
	NTPAddr    string
	CertPath   string
	ServerName string
	Sync       bool
}

func (conf *Config) Check() error {
	return nil
}
