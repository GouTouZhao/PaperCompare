package errs

import (
	"errors"
	"strconv"
	"strings"

	"github.com/GouTouZhao/PaperCompare/tools/errs/stack"
)

const stackSkip = 4

type CodeError interface {
	Code() int
	Msg() string
	Detail() string
	WithDetail(detail string) CodeError
	Error
}

type codeError struct {
	code   int
	msg    string
	detail string
}

func NewCodeError(code int, msg string) CodeError {
	return &codeError{
		code: code,
		msg:  msg,
	}
}

func (e *codeError) Code() int {
	return e.code
}

func (e *codeError) Msg() string {
	return e.msg
}

func (e *codeError) Detail() string {
	return e.detail
}

func (e *codeError) WithDetail(detail string) CodeError {
	var newDetail string
	if e.detail == "" {
		newDetail = detail
	} else {
		newDetail = e.detail + "; " + detail
	}
	return &codeError{
		code:   e.code,
		msg:    e.msg,
		detail: newDetail,
	}
}

func (e *codeError) Wrap() error {
	return stack.New(e, stackSkip)
}

func (e *codeError) clone() *codeError {
	return &codeError{
		code:   e.code,
		msg:    e.msg,
		detail: e.detail,
	}
}

func (e *codeError) WrapMsg(msg string, kv ...any) error {
	newErr := e.clone()
	if msg != "" || len(kv) > 0 {
		detail := toString(msg, kv)
		if newErr.detail == "" {
			newErr.detail = detail
		} else {
			newErr.detail += ", " + detail
		}
	}
	return stack.New(newErr, stackSkip)
}

var Handlers []func(err error) CodeError

func (e *codeError) Is(err error) bool {
	var TempoErr CodeError
	ok := errors.As(Unwrap(err), &TempoErr)
	if !ok {
		if err == nil && e == nil {
			return true
		}
		// 对每个handlers进行筛选
		for i := 0; i < len(Handlers); i++ {
			if TempoErr = Handlers[i](err); TempoErr != nil {
				break
			}
		}
		// 筛选完成
		if TempoErr == nil {
			return false
		}
	}
	if e == nil {
		return false
	}
	code := TempoErr.Code()
	if e.code == code {
		return true
	}
	return e.code == TempoErr.Code()
}

const initialCapacity = 3

func (e *codeError) Error() string {
	v := make([]string, 0, initialCapacity)
	v = append(v, strconv.Itoa(e.code), e.msg)

	if e.detail != "" {
		v = append(v, e.detail)
	}

	return strings.Join(v, " ")
}

// 对于一般错误进行适配
func Unwrap(err error) error {
	for err != nil {
		unwrap, ok := err.(interface {
			error
			Unwrap() error
		})
		if !ok {
			break
		}
		err = unwrap.Unwrap()
		if err == nil {
			return unwrap
		}
	}
	return err
}

func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return stack.New(err, stackSkip)
}

func WrapMsg(err error, msg string, kv ...any) error {
	if err == nil {
		return nil
	}
	err = NewErrorWrapper(err, toString(msg, kv))
	return stack.New(err, stackSkip)
}
