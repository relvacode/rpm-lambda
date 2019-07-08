package setup

import (
	"github.com/aws/aws-sdk-go/aws"
	"log"
	"os"
)

func NewLog(name string) aws.Logger {
	return &Log{
		l: log.New(os.Stderr, name, log.Lshortfile|log.LstdFlags),
	}
}

type Log struct {
	l *log.Logger
}

func (l *Log) Log(v ...interface{}) {
	l.l.Println(v...)
}
