package ator

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/GouTouZhao/PaperCompare/tools/apiresp"
	"github.com/GouTouZhao/PaperCompare/tools/checker"
	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/GouTouZhao/PaperCompare/tools/token"
	"github.com/GouTouZhao/PaperCompare/tools/utils/jsonutil"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"google.golang.org/grpc"
)

type Option[A, B any] struct {
	BindAfter func(*A) error
	RespAfter func(*B) error
}

func RpcContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	uidVal, exists := c.Get("user_id")
	if exists {
		uid, _ := uidVal.(int64)
		email := c.GetString("user_email")
		if uid > 0 {
			ctx = token.InjectUserMetadata(ctx, uid, email)
		}
	}
	return ctx
}

func Call[A, B, C any](c *gin.Context, rpc func(client C, ctx context.Context, rep *A, options ...grpc.CallOption) (*B, error), client C, opts ...*Option[A, B]) {
	req, err := ParseRequestNotCheck[A](c)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}

	for _, opt := range opts {
		if opt.BindAfter == nil {
			continue
		}
		if err := opt.BindAfter(req); err != nil {
			apiresp.GinError(c, err)
			return
		}
	}

	if err := checker.Validate(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}

	resp, err := rpc(client, RpcContext(c), req)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}

	for _, opt := range opts {
		if opt.RespAfter == nil {
			continue
		}
		if err := opt.RespAfter(resp); err != nil {
			apiresp.GinError(c, err)
			return
		}
	}

	apiresp.GinSuccess(c, resp)
}

func ParseIDParam(c *gin.Context, paramName string) (int64, error) {
	raw := c.Param(paramName)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errs.ErrArgs.WrapMsg("invalid id parameter: " + raw)
	}
	return id, nil
}

func ParseRequestNotCheck[T any](c *gin.Context) (*T, error) {
	var req T
	if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete || c.Request.ContentLength == 0 {
		return &req, nil
	}
	if err := c.ShouldBindWith(&req, jsonBind); err != nil {
		return nil, errs.NewCodeError(errs.ArgsError, err.Error())
	}
	return &req, nil
}

type jsonBinding struct{}

var jsonBind binding.Binding = jsonBinding{}

func (jb jsonBinding) Name() string {
	return "json"
}

func (jb jsonBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return errs.WrapMsg(err, "read request body failed", "method", req.Method, "url", req.URL.String())
	}
	if len(body) == 0 {
		return nil
	}
	return errs.Wrap(jb.BindBody(body, obj))
}

func (jb jsonBinding) BindBody(body []byte, obj any) error {
	if err := jsonutil.JsonUnmarshal(body, obj); err != nil {
		return err
	}
	return nil
}
