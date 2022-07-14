package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"ntsc.ac.cn/ta/time-validater/internal/server"
	ccmd "ntsc.ac.cn/tas/tas-commons/pkg/cmd"
)

var serverEnvs struct {
	listener string
}
var serverCmd = &cobra.Command{
	Use:    "server",
	Short:  "TAS time validate server",
	PreRun: _src_prerun,
	Run:    _src_run,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVar(&serverEnvs.listener,
		"bind-addr", "tcp://0.0.0.0:12233",
		"validate tcp listener bind address")
}

func _src_prerun(cmd *cobra.Command, args []string) {
	ccmd.InitGlobalVars()
	var err error
	if err = ccmd.ValidateStringVar(&serverEnvs.listener,
		"bind_addr", true); err != nil {
		logrus.WithField("prefix", "cmd.root").
			Fatalf("check boot var failed: %s", err.Error())
	}
	go func() {
		ccmd.RunWithSysSignal(nil)
	}()
}

func _src_run(cmd *cobra.Command, args []string) {
	s, err := server.NewValidateServer(&server.Config{
		Listener: serverEnvs.listener,
		CertPath: envs.certPath,
	})
	if err != nil {
		logrus.WithField("prefix", "cmd.root").
			Fatalf("failed to create app: %v", err)
	}
	logrus.WithField("prefix", "cmd.root").
		Fatalf("failed to run app: %v", <-s.Start())
}
