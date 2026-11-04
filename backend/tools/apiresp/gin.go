package apiresp

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

const ginApiResponseKey = "Gin_Api_Response_Key"

func ginJson(c *gin.Context, resp *ApiResponse) {
	c.Set(ginApiResponseKey, resp)
	c.JSON(http.StatusOK, resp)
}

func GetApiResponse(c *gin.Context) *ApiResponse {
	resp, ok := c.Get(ginApiResponseKey)
	if !ok {
		return nil
	}
	return resp.(*ApiResponse)
}

func GinError(c *gin.Context, err error) {
	log.Printf("[API ERROR] path: %s, error: %+v", c.Request.URL.Path, err)
	ginJson(c, ParseError(err))
}

func GinSuccess(c *gin.Context, data any) {
	ginJson(c, ApiSuccess(data))
}

