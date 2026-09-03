package designlock

import "errors"

func errorsJoin(errs ...error) error { return errors.Join(errs...) }
