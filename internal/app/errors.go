package app

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

type NotFoundError struct {
	Message string
}

func (e NotFoundError) Error() string {
	return e.Message
}

type PermissionError struct {
	Message string
}

func (e PermissionError) Error() string {
	return e.Message
}

type ExternalError struct {
	Message string
	Err     error
}

func (e ExternalError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return e.Message + ": " + e.Err.Error()
}

func (e ExternalError) Unwrap() error {
	return e.Err
}
