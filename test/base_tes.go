package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/denisbrodbeck/machineid"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func init() {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	formatter := new(prefixed.TextFormatter)
	logrus.SetFormatter(formatter)
}

func TestMachinID(t *testing.T) {
	id, err := machineid.ID()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(id)
}
