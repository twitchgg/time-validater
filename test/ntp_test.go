package test

import (
	"fmt"
	"testing"
	"time"

	"ntsc.ac.cn/ta/time-validater/pkg/tcpntp"
	"ntsc.ac.cn/tas/tas-commons/pkg/rexec"
)

func TestClient(t *testing.T) {
	nc, err := tcpntp.NewNTPClient(&tcpntp.Config{
		Address: "10.25.135.31:12233",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = nc.Open(); err != nil {
		t.Fatal(err)
	}
	defer nc.Close()

	resp, err := nc.Query()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("offset:", resp.ClockOffset)
	local := time.Now()
	localF := local.Format(time.RFC3339Nano)
	fix := local.Add(resp.ClockOffset)
	fixF := fix.Format(time.RFC3339Nano)
	fmt.Println("local time", localF)
	fmt.Println("fix time", fixF)
	fmt.Printf("st-pcie command :cli time_s %04d %02d %02d %02d %02d %02d %d\n",
		fix.Year(), fix.Month(), fix.Day(),
		fix.Hour(), fix.Minute(), fix.Second(), fix.Nanosecond())
	exec, err := rexec.NewExecuter("set_time", "/bin/cli", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = exec.Run(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	fmt.Println("sys time: ", time.Now().Add(time.Second*-37))
}
