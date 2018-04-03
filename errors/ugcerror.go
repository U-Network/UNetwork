package errors

type ugcError struct {
	errmsg    string
	callstack *CallStack
	root      error
	code      ErrCode
}

func (e ugcError) Error() string {
	return e.errmsg
}

func (e ugcError) GetErrCode() ErrCode {
	return e.code
}

func (e ugcError) GetRoot() error {
	return e.root
}

func (e ugcError) GetCallStack() *CallStack {
	return e.callstack
}
