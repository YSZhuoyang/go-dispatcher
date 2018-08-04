package dispatcher

type dispatcherError struct {
	message string
}

func (err *dispatcherError) Error() string {
	return err.message
}

func newError(errMsg string) error {
	return &dispatcherError{
		message: errMsg,
	}
}
