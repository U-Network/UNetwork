package errors

import (
	"errors"
)

const callStackDepth = 10

type DetailError interface {
	error
	ErrCoder
	CallStacker
	GetRoot() error
}

func NewErr(errmsg string) error {
	return errors.New(errmsg)
}

func NewDetailErr(err error, errcode ErrCode, errmsg string) DetailError {
	if err == nil {
		return nil
	}

	ugcerr, ok := err.(ugcError)
	if !ok {
		ugcerr.root = err
		ugcerr.errmsg = err.Error()
		ugcerr.callstack = getCallStack(0, callStackDepth)
		ugcerr.code = errcode

	}
	if errmsg != "" {
		ugcerr.errmsg = errmsg + ": " + ugcerr.errmsg
	}

	return ugcerr
}

func RootErr(err error) error {
	if err, ok := err.(DetailError); ok {
		return err.GetRoot()
	}
	return err
}
