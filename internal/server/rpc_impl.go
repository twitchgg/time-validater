package server

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"ntsc.ac.cn/tas/tas-commons/pkg/pb"
	"ntsc.ac.cn/tas/tas-commons/pkg/rpc"
)

func (s *ValidateServer) Validate(
	stream pb.TimeValidateService_ValidateServer) error {
	var err error
	machineID, err := rpc.GetMachineID(stream.Context())
	if err != nil {
		return rpc.GenerateError(codes.PermissionDenied,
			fmt.Errorf("failed to read machine id: %v", err))
	}
	if cs := s.sm.find(machineID); cs != nil {
		return fmt.Errorf("machine id [%s] existed", machineID)
	}

	cs := newSession(stream, machineID)
	if cs.cronID, err = s.crontab.AddJob("@every 3s", cs); err != nil {
		return rpc.GenerateError(codes.Internal, fmt.Errorf(
			"failed to create crontab job: %v", err))
	}
	s.sm.add(cs)
	logrus.WithField("prefix", "handler_validate").
		Debugf("create validate session: %s", machineID)
	go cs.start()
	for err := range cs.errChan {
		logrus.WithField("prefix", "handler_validate").
			Warnf("session failed: %v", err)
		s.crontab.Remove(cs.cronID)
		s.sm.remove(cs.machineID)
		return nil
	}
	return nil
}

func (s *ValidateServer) Watch(req *pb.HealthCheckRequest, stream pb.Health_WatchServer) error {
	if err := rpc.CheckMachineID(stream.Context(), req.MachineID); err != nil {
		return rpc.GenerateError(codes.PermissionDenied,
			fmt.Errorf("failed to check machin id: %v", err))
	}
	if req.Service != "time-validate-service" {
		return rpc.GenerateArgumentRequiredError("service name")
	}
	for {
		time.Sleep(time.Second * 3)
		if err := stream.Send(&pb.HealthCheckResponse{
			Status: pb.HealthCheckResponse_SERVING,
		}); err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "transport is closing") {
				return nil
			}
			logrus.WithField("prefix", "handler_health_check").
				Warnf("failed to send health check response: %v", err)
			continue
		}
		logrus.WithField("prefix", "rpc.impl").
			Tracef("send health check message to: %s", req.MachineID)
	}
}
