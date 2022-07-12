package server

import (
	"fmt"

	cron "github.com/robfig/cron/v3"
	"google.golang.org/grpc"
	"ntsc.ac.cn/ta-registry/pkg/pb"
	"ntsc.ac.cn/ta-registry/pkg/rpc"
)

type ValidateServer struct {
	conf      *Config
	rpcConf   *rpc.ServerConfig
	rpcServer *rpc.Server
	crontab   *cron.Cron
	sm        *sessionManager
}

func NewValidateServer(conf *Config) (*ValidateServer, error) {
	server := ValidateServer{
		conf:    conf,
		crontab: cron.New(),
		sm:      newSessionManager(),
	}
	var err error
	if conf == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if err = conf.Check(); err != nil {
		return nil, fmt.Errorf("failed to check server config: %v", err)
	}
	if server.rpcConf, err =
		rpc.GenServerRPCConfig(conf.CertPath, conf.Listener); err != nil {
		return nil, fmt.Errorf("failed to generate rpc config: %v", err)
	}
	if server.rpcServer, err = rpc.NewServer(server.rpcConf, []grpc.ServerOption{
		grpc.StreamInterceptor(
			rpc.StreamServerInterceptor(rpc.CertCheckFunc)),
		grpc.UnaryInterceptor(
			rpc.UnaryServerInterceptor(rpc.CertCheckFunc)),
	}, func(g *grpc.Server) {
		pb.RegisterTimeValidateServiceServer(g, &server)
		pb.RegisterHealthServer(g, &server)
	}); err != nil {
		return nil, fmt.Errorf("create grpc server failed: %s", err.Error())
	}

	return &server, nil
}

func (s *ValidateServer) Start() chan error {
	errChan := make(chan error, 1)
	s.crontab.Start()
	go func() {
		err := <-s.rpcServer.Start()
		errChan <- err
	}()
	return errChan
}
