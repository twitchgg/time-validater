package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"ntsc.ac.cn/ta-registry/pkg/pb"
)

const (
	TRAP_URL = "http://127.0.0.1:8787/pushMonitorState"
)

type snmpLog struct {
	ID   string      `json:"id"`
	Data []*snmpData `json:"data"`
}

type snmpData struct {
	OID   string `json:"oid"`
	State int    `json:"state"`
	Type  string `json:"type"`
}

type resultLog struct {
	Result string `json:"result"`
}

type session struct {
	machineID string
	stream    pb.TimeValidateService_ValidateServer
	errChan   chan error
	lastData  *timeData
	cronID    cron.EntryID
}

type timeData struct {
	t1 time.Time
	t2 time.Time
	t3 time.Time
	t4 time.Time
}

func newSession(stream pb.TimeValidateService_ValidateServer,
	machineID string) *session {
	return &session{
		stream:    stream,
		machineID: machineID,
		errChan:   make(chan error),
		lastData:  &timeData{},
	}
}

func (s *session) Run() {
	t1 := timestamppb.Now()
	s.lastData = &timeData{
		t1: t1.AsTime(),
	}
	logrus.WithField("prefix", "session").
		Tracef("send session [%s] t1: %s", s.machineID,
			t1.AsTime().Format(time.RFC3339Nano))
	if err := s.stream.Send(&pb.Request{
		T1: t1,
	}); err != nil {
		logrus.WithField("prefix", "session").Errorf(
			"failed to send data to session [%s]: %v", s.machineID, err)
		s.errChan <- fmt.Errorf(
			"failed to send data to session [%s]: %v", s.machineID, err)
		return
	}
}

func (s *session) start() {
	for {
		resp, err := s.stream.Recv()
		s.lastData.t4 = time.Now()
		if err != nil {
			if err == io.EOF {
				logrus.WithField("prefix", "sessiobn").
					Infof("session [%s] closed", s.machineID)
				return
			}
			s.errChan <- fmt.Errorf(
				"failed to session [%s] recv: %v", s.machineID, err)
			return
		}
		s.lastData.t2 = resp.T2.AsTime()
		s.lastData.t3 = resp.T3.AsTime()
		_t1 := s.lastData.t1.UnixNano()
		_t2 := s.lastData.t2.UnixNano()
		_t3 := s.lastData.t3.UnixNano()
		_t4 := s.lastData.t4.UnixNano()
		offsetValue := ((_t2 - _t1) + (_t3 - _t4)) / 2
		offset := time.Duration(offsetValue)
		offset = offset - time.Duration(time.Second*37)

		logrus.WithField("prefix", "session").
			Tracef("session [%s] offset[%s]", s.machineID, offset)
		go func() {
			log := snmpLog{
				ID: s.machineID,
				Data: []*snmpData{
					{
						OID:   ".1.3.6.1.4.1.326.3.1.1.1",
						Type:  "Counter64",
						State: int(offset),
					},
				},
			}
			logData, err := json.Marshal(&log)
			if err != nil {
				logrus.WithField("prefix", "session").
					Warnf("failed to marshal trap machine [%s] offset[%s]: %v",
						s.machineID, offset, err)
				return
			}
			resp, err := http.Post(TRAP_URL, "application/json", bytes.NewBuffer(logData))
			if err != nil {
				logrus.WithField("prefix", "session").
					Warnf("failed to send trap machine [%s] offset[%s]: %v",
						s.machineID, offset, err)
				return
			}
			var r resultLog
			if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
				logrus.WithField("prefix", "session").
					Warnf("failed to decoder trap response: %v", err)
				return
			}
			logrus.WithField("prefix", "session").
				Tracef("send trap machine [%s] offset[%s] success : %v",
					s.machineID, offset, r.Result)
		}()
	}
}

type sessionManager struct {
	sessions []*session
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		sessions: make([]*session, 0),
	}
}

func (sm *sessionManager) find(machineID string) *session {
	for _, v := range sm.sessions {
		if v.machineID == machineID {
			return v
		}
	}
	return nil
}

func (sm *sessionManager) add(s *session) {
	sm.sessions = append(sm.sessions, s)
}

func (sm *sessionManager) remove(machineID string) {
	ss := make([]*session, 0)
	for _, v := range sm.sessions {
		if v.machineID != machineID {
			ss = append(ss, v)
		}
	}
	sm.sessions = ss
}
