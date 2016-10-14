package dnssd

import (
	"errors"
)

/*
ErrCallback is a closure called when an error has occured in
a dnssd call. err is the error that occurred.
*/
type ErrCallback func(err error)

var errBadFlags error = errors.New("Bad Flags")
