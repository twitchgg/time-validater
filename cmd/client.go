package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"ntsc.ac.cn/ta/time-validater/internal/client"
	ccmd "ntsc.ac.cn/tas/tas-commons/pkg/cmd"
)

var clientEnvs struct {
	endpoint     string
	ntpAddr      string
	mt           bool
	syncFix      int
	SyncInterval int
}
var clientCmd = &cobra.Command{
	Use:    "client",
	Short:  "TAS time validate client",
	PreRun: _client_prerun,
	Run:    _client_run,
}

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.Flags().StringVar(&clientEnvs.endpoint,
		"endpoint", "tcp://127.0.0.1:12233",
		"validate server endpoint")
	clientCmd.Flags().StringVar(&clientEnvs.ntpAddr,
		"ntp-addr", "10.25.135.31:12232",
		"ntp server address")
	clientCmd.Flags().BoolVar(&clientEnvs.mt,
		"sync", false,
		"sync local time")
	clientCmd.Flags().IntVar(&clientEnvs.syncFix,
		"sync-fix", 300,
		"sync fix microsecond")
	clientCmd.Flags().IntVar(&clientEnvs.SyncInterval,
		"sync-interval", 30,
		"sync second")
}

func _client_prerun(cmd *cobra.Command, args []string) {
	ccmd.InitGlobalVars()
	var err error
	if err = ccmd.ValidateStringVar(&clientEnvs.endpoint,
		"endpooint", true); err != nil {
		logrus.WithField("prefix", "cmd.root").
			Fatalf("check boot var failed: %s", err.Error())
	}
	if err = ccmd.ValidateStringVar(&clientEnvs.ntpAddr,
		"ntp_addr", true); err != nil {
		logrus.WithField("prefix", "cmd.root").
			Fatalf("check boot var failed: %s", err.Error())
	}
	go func() {
		ccmd.RunWithSysSignal(nil)
	}()
}

func _client_run(cmd *cobra.Command, args []string) {
	c, err := client.NewValidateClient(&client.Config{
		Endpoint:     clientEnvs.endpoint,
		CertPath:     envs.certPath,
		ServerName:   envs.serverName,
		NTPAddr:      clientEnvs.ntpAddr,
		Sync:         clientEnvs.mt,
		SyncFix:      clientEnvs.syncFix,
		SyncInterval: clientEnvs.SyncInterval,
	})
	if err != nil {
		logrus.WithField("prefix", "cmd.client").
			Fatalf("failed to create client: %v", err)
	}
	logrus.WithField("prefix", "cmd.client").
		Fatalf("failed to run client: %v", <-c.Start())
}
