package server

type Config struct {
	Listener string
	CertPath string
}

func (conf *Config) Check() error {
	return nil
}
