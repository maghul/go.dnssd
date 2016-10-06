package dnssd

import (
	"fmt"
)

type logg func(data ...interface{})

func nulllogg(data ...interface{}) {

}

func stdlog(data ...interface{}) {
	fmt.Println(data)
}

var dnssdlog = nulllogg
var netlog = nulllogg
var qlog = nulllogg
var testlog = nulllogg

func SetLog(name string, logger func(data ...interface{})) {
	switch name {
	case "net":
		netlog = logger
	case "dnssd":
		dnssdlog = logger
	case "q":
		qlog = logger
	case "test":
		testlog = logger
	case "all":
		netlog = logger
		dnssdlog = logger
		qlog = logger
		testlog = logger
	}
}
