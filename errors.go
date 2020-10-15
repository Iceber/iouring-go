// +build linux

package iouring

import "errors"

var (
	ErrIOURingClosed = errors.New("iouring closed")

	ErrRequestCanceled  = errors.New("request is canceled")
	ErrRequestNotFound  = errors.New("request is not found")
	ErrRequestCompleted = errors.New("request has already been completed")

	ErrUnregisteredFile = errors.New("file is unregistered")
)
