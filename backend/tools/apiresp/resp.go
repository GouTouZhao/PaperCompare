package apiresp

import (
	"errors"
	"strconv"
	"strings"

	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"google.golang.org/grpc/status"
)

type ApiResponse struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
	ErrDlt  string `json:"errDlt"`
	Data    any    `json:"data"`
}

func ApiSuccess(data any) *ApiResponse {
	return &ApiResponse{
		ErrCode: 0,
		ErrMsg:  "success",
		ErrDlt:  "",
		Data:    data,
	}
}

func ParseError(err error) *ApiResponse {
	if err == nil {
		return ApiSuccess(nil)
	}

	var codeErr errs.CodeError

	if !errors.As(err, &codeErr) {
		if st, ok := status.FromError(err); ok {
			msg := st.Message()
			if strings.HasPrefix(msg, "Error: ") {
				parts := strings.SplitN(msg[7:], " ", 3)
				if len(parts) >= 3 {
					code, parseErr := strconv.Atoi(parts[0])
					if parseErr == nil {
						detailParts := strings.SplitN(parts[2], " | ", 2)
						errDlt := ""
						if len(detailParts) > 1 {
							errDlt = detailParts[1]
						}
						return &ApiResponse{
							ErrCode: code,
							ErrMsg:  detailParts[0],
							ErrDlt:  errDlt,
							Data:    nil,
						}
					}
				}
			}
			return &ApiResponse{
				ErrCode: errs.ServerInternalError,
				ErrMsg:  msg,
				ErrDlt:  "",
				Data:    nil,
			}
		}

		codeErr = errs.ErrInternalServer.WithDetail(errs.Unwrap(err).Error())
	}

	return &ApiResponse{
		ErrCode: codeErr.Code(),
		ErrMsg:  codeErr.Msg(),
		ErrDlt:  codeErr.Detail(),
		Data:    nil,
	}
}

