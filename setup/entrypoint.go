package setup

import (
	"github.com/aws/aws-sdk-go/aws"
	"os"
)

var setupLog aws.Logger

func init() {
	setupLog = NewLog("entrypoint:setup")
}

func Main(f func() error) {
	err := f()
	if err != nil {
		setupLog.Log(err)
		os.Exit(1)
	}
}
