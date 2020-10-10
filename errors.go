// +build linux

package iouring

import "errors"

var (
	IOURING_ERROR_CANCELED = errors.New("iouring canceled")
)
