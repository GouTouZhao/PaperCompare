package checker

import (
	"github.com/GouTouZhao/PaperCompare/tools/errs"
)

type Checker interface {
	Check() error
}

func Validate(obj any) error {
	if checker, ok := obj.(Checker); ok {
		if err := checker.Check(); err != nil {
			if _, ok := errs.Unwrap(err).(errs.CodeError); ok {
				return err
			}
			return errs.ErrArgs.WrapMsg(err.Error())
		}
	}
	return nil
}
