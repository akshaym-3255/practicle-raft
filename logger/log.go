package logger

import (
	"github.com/sirupsen/logrus"
	"os"
	"fmt"
)

var Log *logrus.Logger

func init() {
	fmt.Println("test")
	Log = logrus.New()
	Log.SetFormatter(&logrus.JSONFormatter{})
	Log.SetLevel(logrus.DebugLevel)
	Log.SetOutput(os.Stdout)
}
