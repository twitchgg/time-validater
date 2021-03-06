package client

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"ntsc.ac.cn/tas/tas-commons/pkg/rexec"
)

func (vc *ValidateClient) _startNTP(errChan chan error) {
	if !vc.conf.Sync {
		return
	}
	if err := vc.ntpClient.Open(); err != nil {
		errChan <- fmt.Errorf("failed to open ntp client")
		return
	}
	vc.crontab.Start()
	interval := fmt.Sprintf("@every %ds", vc.conf.SyncInterval)
	vc.crontab.AddFunc(interval, func() {
		vc._sync(errChan)
	})
}

func (vc *ValidateClient) _sync(errChan chan error) {
	resp, err := vc.ntpClient.Query()
	if err != nil {
		logrus.WithField("preifx", "client.ntp").
			Errorf("failed to query ntp: %v", err)
	}
	logrus.WithField("prefix", "client.ntp").
		Tracef("offset: %s", resp.ClockOffset)
	offset_f64 := math.Abs(float64(resp.ClockOffset))
	conf_f64 := float64(time.Duration(
		time.Millisecond * time.Duration(vc.conf.SyncFix)))
	if offset_f64 < conf_f64 {
		return
	}
	local := time.Now().Add(time.Second * -37)
	fix := local.Add(resp.ClockOffset)
	args := fmt.Sprintf("time_s %04d %02d %02d %02d %02d %02d %d",
		fix.Year(), fix.Month(), fix.Day(),
		fix.Hour(), fix.Minute(), fix.Second(), fix.Nanosecond())
	logrus.WithField("prefix", "client.ntp").
		Tracef("st-pcie command:cli %s", args)
	argsArray := strings.Split(args, " ")
	exec, err := rexec.NewExecuter("set_time", "/bin/cli", argsArray)
	if err != nil {
		logrus.WithField("preifx", "client.ntp").
			Errorf("failed to create set time execute: %v", err)
		return
	}
	result, err := exec.Run()
	if err != nil {
		logrus.WithField("preifx", "client.ntp").
			Errorf("failed to create set time execute: %v %s", err, result)
		return
	}
}
