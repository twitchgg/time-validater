package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"ntsc.ac.cn/ta/time-validater/pkg/tcpntp"
	"ntsc.ac.cn/tas/tas-commons/pkg/pb"
	"ntsc.ac.cn/tas/tas-commons/pkg/rpc"
)

type grpcEntry struct {
	tlsConf *tls.Config
	conn    *grpc.ClientConn
	tsc     pb.TimeValidateServiceClient
	tsvc    pb.TimeValidateService_ValidateClient
	hc      pb.HealthClient
	hwc     pb.Health_WatchClient
}

type ValidateClient struct {
	conf      *Config
	machineID string
	grpcEntry *grpcEntry
	ntpClient *tcpntp.NTPClient
	crontab   *cron.Cron
}

func NewValidateClient(conf *Config) (*ValidateClient, error) {
	if conf == nil {
		return nil, fmt.Errorf("not set trap server config")
	}
	machineID, err := machineid.ID()
	if err != nil {
		return nil, fmt.Errorf("generate machine id failed: %v", err)
	}
	if err := conf.Check(); err != nil {
		return nil, fmt.Errorf("check config failed: %s", err.Error())
	}
	tlsConf, err := rpc.GetTlsConfig(machineID, conf.CertPath, conf.ServerName)
	if err != nil {
		return nil, fmt.Errorf("generate tls config failed: %v", err)
	}
	nc, err := tcpntp.NewNTPClient(&tcpntp.Config{
		Address: conf.NTPAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ntp client: %v", err)
	}
	return &ValidateClient{
		conf:      conf,
		machineID: machineID,
		grpcEntry: &grpcEntry{
			tlsConf: tlsConf,
		},
		ntpClient: nc,
		crontab:   cron.New(),
	}, nil
}

func (vc *ValidateClient) Start() chan error {
	errorChan := make(chan error, 1)
	vc._createRPCClient(errorChan)
	go vc._startHealthChekck(errorChan)
	go vc._startValidate(errorChan)
	go vc._startNTP(errorChan)
	return errorChan
}

func (vc *ValidateClient) _createRPCClient(errChan chan error) {
	var err error
	if vc.grpcEntry.conn != nil {
		vc.grpcEntry.conn.Close()
		vc.grpcEntry.conn = nil
	}
	if vc.grpcEntry.conn, err = rpc.DialRPCConn(&rpc.DialOptions{
		RemoteAddr: vc.conf.Endpoint,
		TLSConfig:  vc.grpcEntry.tlsConf,
	}); err != nil {
		errChan <- fmt.Errorf(
			"dial management grpc connection failed: %v", err)
		return
	}
	vc.grpcEntry.tsc = pb.NewTimeValidateServiceClient(vc.grpcEntry.conn)
	vc.grpcEntry.hc = pb.NewHealthClient(vc.grpcEntry.conn)
	if vc.grpcEntry.tsvc == nil {
		if vc.grpcEntry.tsvc, err = vc.grpcEntry.tsc.Validate(context.Background()); err != nil {
			logrus.WithField("prefix", "trap").
				Errorf("failed to create validate client: %v", err)
			time.Sleep(time.Second)
			vc._createRPCClient(errChan)
		}
	}
	logrus.WithField("prefix", "service.client").Infof(
		"create validate [%s] success", vc.conf.Endpoint)
	if vc.grpcEntry.hwc == nil {
		if vc.grpcEntry.hwc, err = vc.grpcEntry.hc.Watch(context.Background(),
			&pb.HealthCheckRequest{
				Service:   "time-validate-service",
				MachineID: vc.machineID,
			}); err != nil {
			logrus.WithField("prefix", "trap").
				Errorf("failed to create health check client: %v", err)
			time.Sleep(time.Second)
			vc._createRPCClient(errChan)
		}
	}
}

func (vc *ValidateClient) _startHealthChekck(errChan chan error) {
	for {
		resp, err := vc.grpcEntry.hwc.Recv()
		if err != nil || resp == nil {
			if strings.Contains(err.Error(), "EOF") {
				logrus.WithField("prefix", "trap").
					Errorf("time validate service down: %s", vc.conf.Endpoint)
				vc.grpcEntry.tsvc = nil
				vc.grpcEntry.hwc = nil
				time.Sleep(time.Second)
				vc._createRPCClient(errChan)
				continue
			}
			errChan <- fmt.Errorf("rpc failed: %v", err)
			return
		}
		if resp.Status != pb.HealthCheckResponse_SERVING {
			logrus.WithField("prefix", "trap").
				Warnf("snmp trap service status: %s", resp.Status)
		}
	}
}

func (vc *ValidateClient) _startValidate(errChan chan error) {
	for {
		resp, err := vc.grpcEntry.tsvc.Recv()
		t2 := timestamppb.Now()
		if err != nil || resp == nil {
			if strings.Contains(err.Error(), "EOF") {
				logrus.WithField("prefix", "trap").
					Errorf("time validate service down: %s", vc.conf.Endpoint)
				vc.grpcEntry.tsvc = nil
				time.Sleep(time.Second)
				vc._createRPCClient(errChan)
				continue
			}
		}
		t3 := timestamppb.Now()
		if err := vc.grpcEntry.tsvc.Send(&pb.Response{
			MachineID: vc.machineID,
			T2:        t2,
			T3:        t3,
		}); err != nil {
			if strings.Contains(err.Error(), "EOF") {
				logrus.WithField("prefix", "trap").
					Errorf("time validate service [%s] down: %v",
						vc.conf.Endpoint, err)
				vc.grpcEntry.tsvc = nil
				time.Sleep(time.Second)
				vc._createRPCClient(errChan)
				continue
			}
		}
	}
}
