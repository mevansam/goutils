package logger

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/kr/pretty"
	log "github.com/sirupsen/logrus"
)

func Initialize() {

	switch logLevel := os.Getenv("CBS_LOGLEVEL"); logLevel {
	case "debug":
		SetConsoleLogger(log.DebugLevel)
	case "trace":
		SetConsoleLogger(log.TraceLevel)
	default:
		SetConsoleLogger(log.ErrorLevel)
	}
}

func SetConsoleLogger(level log.Level) {

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(level)
}

func DebugMessage(format string, v ...interface{}) {
	if log.IsLevelEnabled(log.DebugLevel) {

		vv := []interface{}{}
		for _, o := range v {
			k := reflect.ValueOf(o).Kind()
			if k == reflect.Struct ||
				k == reflect.Interface ||
				k == reflect.Ptr ||
				k == reflect.Slice ||
				k == reflect.Array ||
				k == reflect.Map {
				vv = append(vv, pretty.Formatter(o))
			} else {
				vv = append(vv, o)
			}
		}

		logMultiLine(fmt.Sprintf(format, vv...), log.Debug)
	}
}

func TraceMessage(format string, v ...interface{}) {
	if log.IsLevelEnabled(log.TraceLevel) {

		vv := []interface{}{}
		for _, o := range v {
			k := reflect.ValueOf(o).Kind()
			if k == reflect.Struct ||
				k == reflect.Interface ||
				k == reflect.Ptr ||
				k == reflect.Slice ||
				k == reflect.Array ||
				k == reflect.Map {
				vv = append(vv, pretty.Formatter(o))
			} else {
				vv = append(vv, o)
			}
		}

		logMultiLine(fmt.Sprintf(format, vv...), log.Trace)
	}
}

func logMultiLine(
	message string,
	logFunc func(args ...interface{})) {

	now := time.Now()
	s := bufio.NewScanner(strings.NewReader(message))
	for s.Scan() {
		log.WithTime(now)
		logFunc(s.Text())
	}
}
