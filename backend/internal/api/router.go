package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	protocompare "github.com/GouTouZhao/PaperCompare/internal/protocol/compare"
	protopaper "github.com/GouTouZhao/PaperCompare/internal/protocol/paper"
	protosdkws "github.com/GouTouZhao/PaperCompare/internal/protocol/sdkws"
	protouser "github.com/GouTouZhao/PaperCompare/internal/protocol/user"
	"github.com/GouTouZhao/PaperCompare/tools/apiresp"
	"github.com/GouTouZhao/PaperCompare/tools/ator"
	"github.com/GouTouZhao/PaperCompare/tools/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/GouTouZhao/PaperCompare/tools/token"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, x-token, X-Token")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Disposition, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func AuthMiddleware(required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken := c.GetHeader("Authorization")
		if rawToken == "" {
			rawToken = c.GetHeader("x-token")
		}
		if rawToken == "" {
			rawToken = c.Query("token")
		}
		claims, err := token.ParseAccessToken(rawToken)
		if err != nil {
			if required {
				apiresp.GinError(c, errs.ErrToken.WrapMsg("access_token invalid or expired"))
				c.Abort()
				return
			}
			c.Next()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}

type GatewayHandlers struct {
	cfg           *Config
	dataDir       string
	userClient    protouser.UserClient
	paperClient   protopaper.PaperClient
	compareClient protocompare.CompareClient
}

func newGinRouter(ctx context.Context, client discovery.SvcDiscoveryRegistry, cfg *Config) (*gin.Engine, error) {
	userConn, err := client.GetConn(ctx, cfg.Discovery.RpcService.User)
	if err != nil {
		return nil, err
	}
	paperConn, err := client.GetConn(ctx, cfg.Discovery.RpcService.Paper)
	if err != nil {
		return nil, err
	}
	compareConn, err := client.GetConn(ctx, cfg.Discovery.RpcService.Compare)
	if err != nil {
		return nil, err
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(Cors())
	r.NoRoute(Cors())
	r.NoMethod(Cors())

	if cfg.API.Api.CompressionLevel > 0 {
		r.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	dataDir := cfg.Storage.DataDir
	if envDir := os.Getenv("STORAGE_DATA_DIR"); envDir != "" {
		dataDir = envDir
	}
	if dataDir == "" {
		dataDir = "./storage"
	}
	_ = os.MkdirAll(filepath.Join(dataDir, "papers"), 0755)

	h := &GatewayHandlers{
		cfg:           cfg,
		dataDir:       dataDir,
		userClient:    protouser.NewUserClient(userConn),
		paperClient:   protopaper.NewPaperClient(paperConn),
		compareClient: protocompare.NewCompareClient(compareConn),
	}

	// Health Endpoint
	r.GET("/api/health", func(c *gin.Context) {
		apiresp.GinSuccess(c, gin.H{"status": "ok", "time": time.Now().Unix()})
	})

	// Auth routes (public)
	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/register", func(c *gin.Context) {
			ator.Call(c, protouser.UserClient.Register, h.userClient)
		})
		authGroup.POST("/login", func(c *gin.Context) {
			ator.Call(c, protouser.UserClient.Login, h.userClient)
		})
		authGroup.POST("/refresh", func(c *gin.Context) {
			ator.Call(c, protouser.UserClient.Refresh, h.userClient)
		})
		authGroup.POST("/logout", func(c *gin.Context) {
			ator.Call(c, protouser.UserClient.Logout, h.userClient)
		})
		authGroup.GET("/me", AuthMiddleware(true), func(c *gin.Context) {
			ator.Call(c, protouser.UserClient.GetMe, h.userClient)
		})
	}

	// Protected API routes
	apiGroup := r.Group("/api")
	apiGroup.Use(AuthMiddleware(true))
	{
		// Home / Comparison History Overview
		apiGroup.GET("/dashboard/overview", h.GetDashboardFull)
		apiGroup.GET("/dashboard/recent-comparisons", h.GetDashboardRecentComparisons)

		// Papers (for comparison input, upload, and PDF page evidence inspection)
		apiGroup.POST("/papers/upload", h.UploadPaper)
		apiGroup.GET("/papers", h.ListPapers)
		apiGroup.GET("/papers/:id", h.GetPaperDetail)
		apiGroup.DELETE("/papers/:id", h.DeletePaper)
		apiGroup.GET("/papers/:id/file", h.StreamPaperFile)
		apiGroup.GET("/papers/:id/pages", h.GetPaperPages)
		apiGroup.POST("/papers/:id/analyze", h.TriggerAnalyze)
		apiGroup.GET("/papers/:id/analysis", h.GetPaperAnalysis)
		apiGroup.GET("/jobs/:id", h.GetJobStatus)

		// Multi-paper Comparison Workspace
		apiGroup.GET("/comparison-templates", func(c *gin.Context) {
			ator.Call(c, protocompare.CompareClient.ListTemplates, h.compareClient)
		})
		apiGroup.POST("/comparison-templates", func(c *gin.Context) {
			ator.Call(c, protocompare.CompareClient.CreateTemplate, h.compareClient)
		})
		apiGroup.DELETE("/comparison-templates/:id", h.DeleteTemplate)

		apiGroup.GET("/comparisons", h.ListComparisons)
		apiGroup.POST("/comparisons", func(c *gin.Context) {
			ator.Call(c, protocompare.CompareClient.CreateComparison, h.compareClient)
		})
		apiGroup.GET("/comparisons/:id", h.GetComparison)
		apiGroup.PUT("/comparisons/:id", h.UpdateComparison)
		apiGroup.DELETE("/comparisons/:id", h.DeleteComparison)
		apiGroup.PUT("/comparisons/:id/cell", h.UpdateComparisonCell)
		apiGroup.PUT("/comparisons/:id/results", h.BatchUpdateCells)
		apiGroup.POST("/comparisons/:id/auto-fill", h.AutoFillComparison)
		apiGroup.POST("/comparisons/:id/analyze", h.AnalyzeComparison)
		apiGroup.PUT("/comparisons/:id/diff-analysis", h.UpdateDiffAnalysis)
		apiGroup.GET("/comparisons/:id/export", h.ExportComparison)
	}

	return r, nil
}

// ===== Handler Implementations =====

func (h *GatewayHandlers) GetDashboardFull(c *gin.Context) {
	rpcCtx := ator.RpcContext(c)
	overview, err := h.paperClient.GetDashboardOverview(rpcCtx, &protosdkws.Empty{})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	compsResp, _ := h.compareClient.ListComparisons(rpcCtx, &protocompare.ListComparisonsReq{Page: 1, PageSize: 20})
	userResp, _ := h.userClient.GetMe(rpcCtx, &protosdkws.Empty{})

	var recentComps []*protocompare.ComparisonItem
	if compsResp != nil {
		recentComps = compsResp.List
	}

	apiresp.GinSuccess(c, gin.H{
		"overview":           overview,
		"recent_comparisons": recentComps,
		"user":               userResp,
	})
}

func (h *GatewayHandlers) GetDashboardRecentComparisons(c *gin.Context) {
	resp, err := h.compareClient.ListComparisons(ator.RpcContext(c), &protocompare.ListComparisonsReq{Page: 1, PageSize: 20})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) UploadPaper(c *gin.Context) {
	uid := c.GetInt64("user_id")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		apiresp.GinError(c, errs.ErrArgs.WrapMsg("请选择要上传的 PDF 文件"))
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".pdf" {
		apiresp.GinError(c, errs.ErrArgs.WrapMsg("仅支持上传 .pdf 格式文件"))
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > 100*1024*1024 {
		apiresp.GinError(c, errs.ErrArgs.WrapMsg("PDF 文件大小需在 1B 到 100MB 之间"))
		return
	}

	srcFile, err := fileHeader.Open()
	if err != nil {
		apiresp.GinError(c, errs.ErrInternalServer.WrapMsg("无法读取上传文件"))
		return
	}
	defer srcFile.Close()

	// Verify PDF magic header (%PDF-)
	magic := make([]byte, 5)
	if n, err := srcFile.Read(magic); err != nil || n < 5 || !strings.HasPrefix(string(magic), "%PDF-") {
		apiresp.GinError(c, errs.ErrArgs.WrapMsg("文件内容校验失败：不是合法的 PDF 文档 (%PDF- header missing)"))
		return
	}
	_, _ = srcFile.Seek(0, io.SeekStart)

	userDir := filepath.Join(h.dataDir, "papers", fmt.Sprintf("user_%d", uid))
	_ = os.MkdirAll(userDir, 0755)

	safeName := fmt.Sprintf("paper_%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	destPath := filepath.Join(userDir, safeName)
	dstFile, err := os.Create(destPath)
	if err != nil {
		apiresp.GinError(c, errs.ErrInternalServer.WrapMsg("保存 PDF 文件失败"))
		return
	}

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(dstFile, hasher), srcFile)
	dstFile.Close()
	if copyErr != nil {
		_ = os.Remove(destPath)
		apiresp.GinError(c, errs.ErrInternalServer.WrapMsg("写入 PDF 文件失败"))
		return
	}
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	forceUpload := c.PostForm("force_upload") == "true"
	autoAnalyze := c.DefaultPostForm("auto_analyze", "true") != "false"
	title := c.PostForm("title")

	resp, err := h.paperClient.RegisterPaperUpload(ator.RpcContext(c), &protopaper.RegisterPaperUploadReq{
		Title:            title,
		FilePath:         destPath,
		OriginalFilename: fileHeader.Filename,
		FileSize:         written,
		MimeType:         "application/pdf",
		Sha256:           shaHex,
		ForceUpload:      forceUpload,
		AutoAnalyze:      autoAnalyze,
	})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	if resp.DuplicateFound {
		_ = os.Remove(destPath)
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) ListPapers(c *gin.Context) {
	yearFrom, _ := strconv.Atoi(c.Query("year_from"))
	yearTo, _ := strconv.Atoi(c.Query("year_to"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	resp, err := h.paperClient.ListPapers(ator.RpcContext(c), &protopaper.ListPapersReq{
		Keyword:     c.Query("keyword"),
		Author:      c.Query("author"),
		YearFrom:    int32(yearFrom),
		YearTo:      int32(yearTo),
		ParseStatus: c.Query("parse_status"),
		SortBy:      c.Query("sort_by"),
		SortOrder:   c.Query("sort_order"),
		Page:        int32(page),
		PageSize:    int32(pageSize),
	})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) GetPaperDetail(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.paperClient.GetPaperDetail(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) DeletePaper(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.paperClient.DeletePaper(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) StreamPaperFile(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.paperClient.GetPaperFilePath(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	if _, statErr := os.Stat(resp.FilePath); statErr != nil {
		apiresp.GinError(c, errs.ErrRecordNotFound.WrapMsg("PDF 原文件不存在"))
		return
	}
	if c.Query("download") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", resp.OriginalFilename))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", resp.OriginalFilename))
	}
	c.Header("Content-Type", "application/pdf")
	http.ServeFile(c.Writer, c.Request, resp.FilePath)
}

func (h *GatewayHandlers) GetPaperPages(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.paperClient.GetPaperPages(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) TriggerAnalyze(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.paperClient.TriggerParseAndAnalyze(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) GetPaperAnalysis(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.paperClient.GetPaperAnalysis(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) GetJobStatus(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.paperClient.GetJobStatus(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) DeleteTemplate(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.compareClient.DeleteTemplate(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) ListComparisons(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resp, err := h.compareClient.ListComparisons(ator.RpcContext(c), &protocompare.ListComparisonsReq{
		Keyword:  c.Query("keyword"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) GetComparison(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.compareClient.GetComparison(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) UpdateComparison(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	ator.Call(c, protocompare.CompareClient.UpdateComparison, h.compareClient, &ator.Option[protocompare.UpdateComparisonReq, protocompare.ComparisonItem]{
		BindAfter: func(req *protocompare.UpdateComparisonReq) error {
			req.Id = id
			return nil
		},
	})
}

func (h *GatewayHandlers) DeleteComparison(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.compareClient.DeleteComparison(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) UpdateComparisonCell(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	ator.Call(c, protocompare.CompareClient.UpdateCell, h.compareClient, &ator.Option[protocompare.UpdateCellReq, protocompare.ComparisonCell]{
		BindAfter: func(req *protocompare.UpdateCellReq) error {
			req.ComparisonId = id
			return nil
		},
	})
}

func (h *GatewayHandlers) BatchUpdateCells(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	ator.Call(c, protocompare.CompareClient.BatchUpdateCells, h.compareClient, &ator.Option[protocompare.BatchUpdateCellsReq, protocompare.ComparisonItem]{
		BindAfter: func(req *protocompare.BatchUpdateCellsReq) error {
			req.ComparisonId = id
			return nil
		},
	})
}

func (h *GatewayHandlers) AutoFillComparison(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	ator.Call(c, protocompare.CompareClient.AutoFill, h.compareClient, &ator.Option[protocompare.AutoFillReq, protocompare.ComparisonItem]{
		BindAfter: func(req *protocompare.AutoFillReq) error {
			req.ComparisonId = id
			return nil
		},
	})
}

func (h *GatewayHandlers) AnalyzeComparison(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	resp, err := h.compareClient.AnalyzeDifferences(ator.RpcContext(c), &protosdkws.IDReq{Id: id})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (h *GatewayHandlers) UpdateDiffAnalysis(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	ator.Call(c, protocompare.CompareClient.UpdateDiffAnalysis, h.compareClient, &ator.Option[protocompare.UpdateDiffAnalysisReq, protocompare.ComparisonItem]{
		BindAfter: func(req *protocompare.UpdateDiffAnalysisReq) error {
			req.ComparisonId = id
			return nil
		},
	})
}

func (h *GatewayHandlers) ExportComparison(c *gin.Context) {
	id, err := ator.ParseIDParam(c, "id")
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	format := c.DefaultQuery("format", "markdown")
	resp, err := h.compareClient.ExportComparison(ator.RpcContext(c), &protocompare.ExportComparisonReq{
		ComparisonId: id,
		Format:       format,
	})
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", resp.Filename))
	c.Data(http.StatusOK, resp.ContentType, resp.Data)
}
