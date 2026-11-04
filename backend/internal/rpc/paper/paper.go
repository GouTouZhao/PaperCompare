package paper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	protopaper "github.com/GouTouZhao/PaperCompare/internal/protocol/paper"
	protosdkws "github.com/GouTouZhao/PaperCompare/internal/protocol/sdkws"
	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/pkg/common/storage/database/model"
	"github.com/GouTouZhao/PaperCompare/pkg/dbbuild"
	"github.com/GouTouZhao/PaperCompare/tools/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/GouTouZhao/PaperCompare/tools/llm"
	"github.com/GouTouZhao/PaperCompare/tools/pdfparser"
	"github.com/GouTouZhao/PaperCompare/tools/snowflake"
	"github.com/GouTouZhao/PaperCompare/tools/token"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type Config struct {
	config.ALLConfig
	ConfigPath config.Path
	Index      config.Index
}

type PaperServer struct {
	protopaper.UnimplementedPaperServer
	config    *Config
	db        *gorm.DB
	idGen     *snowflake.Snowflake
	llmClient *llm.Client
	dataDir   string
}

func Start(ctx context.Context, cfg *Config, client discovery.SvcDiscoveryRegistry, server grpc.ServiceRegistrar) error {
	dbB := dbbuild.NewBuilder(&cfg.Postgres)
	sqlcli, err := dbB.Postgres(ctx)
	if err != nil {
		return err
	}

	if err := model.AutoMigrate(sqlcli); err != nil {
		return err
	}

	idGen, err := snowflake.NewPodIPHashSnowflake()
	if err != nil {
		return err
	}

	dataDir := cfg.Storage.DataDir
	if envDir := os.Getenv("STORAGE_DATA_DIR"); envDir != "" {
		dataDir = envDir
	}
	if dataDir == "" {
		dataDir = "./storage"
	}
	_ = os.MkdirAll(filepath.Join(dataDir, "papers"), 0755)

	llmCli := llm.NewClient(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)

	srv := &PaperServer{
		config:    cfg,
		db:        sqlcli,
		idGen:     idGen,
		llmClient: llmCli,
		dataDir:   dataDir,
	}

	srv.seedDemoPapers(ctx)

	protopaper.RegisterPaperServer(server, srv)
	return nil
}

func marshalStringSlice(items []string) string {
	if items == nil {
		items = []string{}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func unmarshalStringSlice(raw string) []string {
	var out []string
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func marshalAnalysisField(f *protopaper.AnalysisField) string {
	if f == nil {
		f = &protopaper.AnalysisField{Value: "未报告", SourceType: "not_reported", VerificationStatus: "unverified"}
	}
	b, _ := json.Marshal(f)
	return string(b)
}

func unmarshalAnalysisField(raw string) *protopaper.AnalysisField {
	var f protopaper.AnalysisField
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &f)
	}
	if f.SourceType == "" {
		f.SourceType = "explicit"
	}
	if f.VerificationStatus == "" {
		f.VerificationStatus = "unverified"
	}
	return &f
}

func (s *PaperServer) enrichPaperItem(ctx context.Context, p *model.Paper) *protopaper.PaperItem {
	item := &protopaper.PaperItem{
		Id:              p.ID,
		UserId:          p.UserID,
		Title:           p.Title,
		Abstract:        p.Abstract,
		Authors:         unmarshalStringSlice(p.Authors),
		PublicationYear: p.PublicationYear,
		Journal:         p.Journal,
		Doi:             p.DOI,
		Language:        p.Language,
		PaperType:       p.PaperType,
		Keywords:        unmarshalStringSlice(p.Keywords),
		PublicationUrl:  p.PublicationURL,
		ReadingStatus:   p.ReadingStatus,
		ParseStatus:     p.ParseStatus,
		IsFavorite:      p.IsFavorite,
		CreatedAt:       p.CreatedAt.Unix(),
		UpdatedAt:       p.UpdatedAt.Unix(),
	}

	var pf model.PaperFile
	if err := s.db.WithContext(ctx).Where("paper_id = ?", p.ID).Order("id desc").First(&pf).Error; err == nil {
		item.FileName = pf.OriginalFilename
		item.FileSize = pf.FileSize
		item.PageCount = pf.PageCount
		item.Sha256 = pf.SHA256
	}

	return item
}

func (s *PaperServer) RegisterPaperUpload(ctx context.Context, req *protopaper.RegisterPaperUploadReq) (*protopaper.RegisterPaperUploadResp, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.FilePath == "" {
		return nil, errs.ErrArgs.WrapMsg("file_path is required")
	}

	if !req.ForceUpload && req.Sha256 != "" {
		var existingFile model.PaperFile
		if err := s.db.WithContext(ctx).Where("user_id = ? AND sha256 = ?", uid, req.Sha256).First(&existingFile).Error; err == nil {
			var existingPaper model.Paper
			if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", existingFile.PaperID, uid).First(&existingPaper).Error; err == nil {
				return &protopaper.RegisterPaperUploadResp{
					DuplicateFound: true,
					DuplicatePaper: s.enrichPaperItem(ctx, &existingPaper),
				}, nil
			}
		}
	}

	paperID, _ := s.idGen.NextID()
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSuffix(req.OriginalFilename, filepath.Ext(req.OriginalFilename))
	}

	newPaper := &model.Paper{
		ID:              paperID,
		UserID:          uid,
		Title:           title,
		Authors:         marshalStringSlice([]string{}),
		PublicationYear: int32(time.Now().Year()),
		Language:        "English",
		PaperType:       "Research Paper",
		Keywords:        marshalStringSlice([]string{}),
		ReadingStatus:   "unread",
		ParseStatus:     "uploaded",
	}

	if err := s.db.WithContext(ctx).Create(newPaper).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg("创建论文记录失败: " + err.Error())
	}

	fileID, _ := s.idGen.NextID()
	pf := &model.PaperFile{
		ID:               fileID,
		PaperID:          paperID,
		UserID:           uid,
		FilePath:         req.FilePath,
		OriginalFilename: req.OriginalFilename,
		FileSize:         req.FileSize,
		MimeType:         req.MimeType,
		SHA256:           req.Sha256,
		PageCount:        0,
	}
	if err := s.db.WithContext(ctx).Create(pf).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg("保存文件元数据失败: " + err.Error())
	}

	jobItem, _ := s.startParseJob(uid, paperID, req.AutoAnalyze)

	// Wait briefly if fast local parse finishes quickly so initial upload response already has metadata
	time.Sleep(350 * time.Millisecond)
	var updatedPaper model.Paper
	_ = s.db.WithContext(ctx).Where("id = ?", paperID).First(&updatedPaper).Error
	var updatedJob model.ProcessingJob
	if jobItem != nil {
		_ = s.db.WithContext(ctx).Where("id = ?", jobItem.Id).First(&updatedJob).Error
		jobItem = toProtoJob(&updatedJob)
	}

	return &protopaper.RegisterPaperUploadResp{
		DuplicateFound: false,
		Paper:          s.enrichPaperItem(ctx, &updatedPaper),
		Job:            jobItem,
	}, nil
}

func toProtoJob(j *model.ProcessingJob) *protopaper.ProcessingJobItem {
	if j == nil || j.ID == 0 {
		return nil
	}
	var fin int64
	if !j.FinishedAt.IsZero() {
		fin = j.FinishedAt.Unix()
	}
	return &protopaper.ProcessingJobItem{
		Id:           j.ID,
		PaperId:      j.PaperID,
		JobType:      j.JobType,
		Status:       j.Status,
		Progress:     j.Progress,
		ErrorMessage: j.ErrorMessage,
		CreatedAt:    j.CreatedAt.Unix(),
		FinishedAt:   fin,
	}
}

func (s *PaperServer) startParseJob(userID, paperID int64, runLLM bool) (*protopaper.ProcessingJobItem, error) {
	jobID, _ := s.idGen.NextID()
	job := &model.ProcessingJob{
		ID:         jobID,
		UserID:     userID,
		PaperID:    paperID,
		JobType:    "parse_and_extract",
		Status:     "queued",
		Progress:   5,
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
	}
	if err := s.db.Create(job).Error; err != nil {
		return nil, err
	}

	go s.executeParseAndAnalyze(jobID, userID, paperID, runLLM)
	return toProtoJob(job), nil
}

func (s *PaperServer) executeParseAndAnalyze(jobID, userID, paperID int64, runLLM bool) {
	bgCtx := context.Background()
	s.db.Model(&model.ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":   "parsing",
		"progress": 20,
	})
	s.db.Model(&model.Paper{}).Where("id = ?", paperID).Update("parse_status", "parsing")

	var pf model.PaperFile
	if err := s.db.Where("paper_id = ?", paperID).Order("id desc").First(&pf).Error; err != nil {
		s.failJob(jobID, paperID, "未找到关联的 PDF 文件")
		return
	}

	parsedDoc, err := pdfparser.ParsePDFFile(pf.FilePath)
	if err != nil {
		s.failJob(jobID, paperID, "PDF 解析失败: "+err.Error())
		return
	}

	s.db.Model(&model.PaperFile{}).Where("id = ?", pf.ID).Update("page_count", parsedDoc.PageCount)

	// Save pages
	s.db.Where("paper_id = ?", paperID).Delete(&model.PaperPage{})
	pageMap := make(map[int]string)
	for _, pg := range parsedDoc.Pages {
		pgID, _ := s.idGen.NextID()
		s.db.Create(&model.PaperPage{
			ID:         pgID,
			PaperID:    paperID,
			PageNumber: int32(pg.PageNumber),
			RawText:    pg.RawText,
			CleanText:  pg.CleanText,
		})
		pageMap[pg.PageNumber] = pg.CleanText
	}

	// Save sections
	s.db.Where("paper_id = ?", paperID).Delete(&model.PaperSection{})
	for _, sec := range parsedDoc.Sections {
		secID, _ := s.idGen.NextID()
		s.db.Create(&model.PaperSection{
			ID:           secID,
			PaperID:      paperID,
			SectionTitle: sec.SectionTitle,
			SectionType:  sec.SectionType,
			StartPage:    int32(sec.StartPage),
			EndPage:      int32(sec.EndPage),
			Content:      sec.Content,
			SectionOrder: int32(sec.SectionOrder),
		})
	}

	var p model.Paper
	s.db.Where("id = ?", paperID).First(&p)
	updates := map[string]any{}
	if p.Abstract == "" && parsedDoc.Abstract != "" {
		updates["abstract"] = parsedDoc.Abstract
	}
	if p.DOI == "" && parsedDoc.DOI != "" {
		updates["doi"] = parsedDoc.DOI
	}
	if (p.Title == "" || strings.HasSuffix(strings.ToLower(p.Title), ".pdf")) && parsedDoc.Title != "" {
		updates["title"] = parsedDoc.Title
	}
	if len(updates) > 0 {
		s.db.Model(&model.Paper{}).Where("id = ?", paperID).Updates(updates)
		s.db.Where("id = ?", paperID).First(&p)
	}

	s.db.Model(&model.ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":   "extracting",
		"progress": 60,
	})
	s.db.Model(&model.Paper{}).Where("id = ?", paperID).Update("parse_status", "extracting")

	if runLLM {
		extCtx, cancel := context.WithTimeout(bgCtx, 70*time.Second)
		defer cancel()
		extracted, llmErr := s.llmClient.ExtractPaperInfo(extCtx, pageMap, p.Title, p.Abstract)
		if llmErr == nil && extracted != nil {
			s.saveLLMExtraction(paperID, &p, extracted)
			s.db.Model(&model.Paper{}).Where("id = ?", paperID).Update("parse_status", "completed")
			s.db.Model(&model.ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
				"status":      "completed",
				"progress":    100,
				"finished_at": time.Now(),
			})
			return
		}
		// Fallback heuristic analysis when LLM fails or times out
		s.saveHeuristicAnalysis(paperID, &p, parsedDoc)
		s.db.Model(&model.Paper{}).Where("id = ?", paperID).Update("parse_status", "partial")
		s.db.Model(&model.ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
			"status":        "partial",
			"progress":      100,
			"error_message": fmt.Sprintf("已完成 PDF 章节提取，AI 深度提取回退为规则提取 (%v)", llmErr),
			"finished_at":   time.Now(),
		})
		return
	}

	s.saveHeuristicAnalysis(paperID, &p, parsedDoc)
	s.db.Model(&model.Paper{}).Where("id = ?", paperID).Update("parse_status", "completed")
	s.db.Model(&model.ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":      "completed",
		"progress":    100,
		"finished_at": time.Now(),
	})
}

func convertLLMField(f llm.ExtractedField) *protopaper.AnalysisField {
	val := strings.TrimSpace(f.Value)
	st := f.SourceType
	if val == "" {
		val = "未报告"
		st = "not_reported"
	}
	if st == "" {
		st = "explicit"
	}
	out := &protopaper.AnalysisField{
		Value:              val,
		SourceType:         st,
		VerificationStatus: "unverified",
	}
	for _, ev := range f.Evidence {
		pg := ev.Page
		if pg <= 0 {
			pg = 1
		}
		out.Evidence = append(out.Evidence, &protosdkws.Evidence{
			Page:    pg,
			Quote:   ev.Quote,
			Section: ev.Section,
		})
	}
	return out
}

func (s *PaperServer) saveLLMExtraction(paperID int64, p *model.Paper, ext *llm.PaperExtractionResult) {
	metaUpdates := map[string]any{}
	if ext.Title != "" && (p.Title == "" || len(p.Title) < 6) {
		metaUpdates["title"] = ext.Title
	}
	if len(ext.Authors) > 0 && len(unmarshalStringSlice(p.Authors)) == 0 {
		metaUpdates["authors"] = marshalStringSlice(ext.Authors)
	}
	if ext.PublicationYear >= 1990 && ext.PublicationYear <= 2030 {
		metaUpdates["publication_year"] = ext.PublicationYear
	}
	if ext.Journal != "" && p.Journal == "" {
		metaUpdates["journal"] = ext.Journal
	}
	if ext.DOI != "" && p.DOI == "" {
		metaUpdates["doi"] = ext.DOI
	}
	if ext.Abstract != "" && p.Abstract == "" {
		metaUpdates["abstract"] = ext.Abstract
	}
	if len(ext.Keywords) > 0 && len(unmarshalStringSlice(p.Keywords)) == 0 {
		metaUpdates["keywords"] = marshalStringSlice(ext.Keywords)
	}
	if len(metaUpdates) > 0 {
		s.db.Model(&model.Paper{}).Where("id = ?", paperID).Updates(metaUpdates)
	}

	var existing model.PaperAnalysis
	err := s.db.Where("paper_id = ?", paperID).First(&existing).Error
	analysisID := existing.ID
	version := existing.AnalysisVersion + 1
	if err != nil {
		analysisID, _ = s.idGen.NextID()
		version = 1
	}

	record := &model.PaperAnalysis{
		ID:                analysisID,
		PaperID:           paperID,
		ResearchProblem:   marshalAnalysisField(convertLLMField(ext.ResearchProblem)),
		ResearchObjective: marshalAnalysisField(convertLLMField(ext.ResearchObjective)),
		Contributions:     marshalAnalysisField(convertLLMField(ext.Contributions)),
		Methodology:       marshalAnalysisField(convertLLMField(ext.Methodology)),
		Datasets:          marshalAnalysisField(convertLLMField(ext.Datasets)),
		ExpSettings:       marshalAnalysisField(convertLLMField(ext.ExpSettings)),
		Metrics:           marshalAnalysisField(convertLLMField(ext.Metrics)),
		Results:           marshalAnalysisField(convertLLMField(ext.Results)),
		Limitations:       marshalAnalysisField(convertLLMField(ext.Limitations)),
		FutureWork:        marshalAnalysisField(convertLLMField(ext.FutureWork)),
		AnalysisVersion:   version,
		UpdatedAt:         time.Now(),
	}
	if err == nil {
		s.db.Model(&existing).Updates(record)
	} else {
		s.db.Create(record)
	}
}

func (s *PaperServer) saveHeuristicAnalysis(paperID int64, p *model.Paper, doc *pdfparser.ParsedDocument) {
	var existing model.PaperAnalysis
	if err := s.db.Where("paper_id = ?", paperID).First(&existing).Error; err == nil {
		return
	}

	snippet := func(secType string, fallbackPage int32) *protopaper.AnalysisField {
		for _, sec := range doc.Sections {
			if strings.EqualFold(sec.SectionType, secType) && len(sec.Content) > 20 {
				text := sec.Content
				if len(text) > 360 {
					text = text[:360] + "..."
				}
				return &protopaper.AnalysisField{
					Value:              text,
					SourceType:         "explicit",
					VerificationStatus: "unverified",
					Evidence: []*protosdkws.Evidence{
						{Page: int32(sec.StartPage), Quote: text, Section: sec.SectionTitle},
					},
				}
			}
		}
		return &protopaper.AnalysisField{
			Value:              "待进一步核实或点击 AI 深度提取",
			SourceType:         "unverified",
			VerificationStatus: "unverified",
			Evidence:           []*protosdkws.Evidence{{Page: fallbackPage, Quote: p.Title, Section: "Overview"}},
		}
	}

	analysisID, _ := s.idGen.NextID()
	rec := &model.PaperAnalysis{
		ID:                analysisID,
		PaperID:           paperID,
		ResearchProblem:   marshalAnalysisField(snippet("Introduction", 1)),
		ResearchObjective: marshalAnalysisField(snippet("Abstract", 1)),
		Contributions:     marshalAnalysisField(snippet("Introduction", 1)),
		Methodology:       marshalAnalysisField(snippet("Methodology", 2)),
		Datasets:          marshalAnalysisField(snippet("Experiments", 3)),
		ExpSettings:       marshalAnalysisField(snippet("Experiments", 3)),
		Metrics:           marshalAnalysisField(snippet("Results", 4)),
		Results:           marshalAnalysisField(snippet("Results", 4)),
		Limitations:       marshalAnalysisField(snippet("Conclusion", int32(doc.PageCount))),
		FutureWork:        marshalAnalysisField(snippet("Conclusion", int32(doc.PageCount))),
		AnalysisVersion:   1,
	}
	s.db.Create(rec)
}

func (s *PaperServer) failJob(jobID, paperID int64, msg string) {
	s.db.Model(&model.ProcessingJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        "failed",
		"progress":      100,
		"error_message": msg,
		"finished_at":   time.Now(),
	})
	s.db.Model(&model.Paper{}).Where("id = ?", paperID).Update("parse_status", "failed")
}

func (s *PaperServer) ListPapers(ctx context.Context, req *protopaper.ListPapersReq) (*protopaper.ListPapersResp, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	q := s.db.WithContext(ctx).Model(&model.Paper{}).Where("papers.user_id = ?", uid)

	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(papers.title LIKE ? OR papers.abstract LIKE ? OR papers.doi LIKE ? OR papers.keywords LIKE ? OR papers.journal LIKE ?)",
			like, like, like, like, like)
	}
	if au := strings.TrimSpace(req.Author); au != "" {
		q = q.Where("papers.authors LIKE ?", "%"+au+"%")
	}
	if req.YearFrom > 0 {
		q = q.Where("papers.publication_year >= ?", req.YearFrom)
	}
	if req.YearTo > 0 {
		q = q.Where("papers.publication_year <= ?", req.YearTo)
	}
	if req.ParseStatus != "" && req.ParseStatus != "all" {
		q = q.Where("papers.parse_status = ?", req.ParseStatus)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}

	orderClause := "papers.created_at DESC"
	dir := "DESC"
	if strings.EqualFold(req.SortOrder, "asc") {
		dir = "ASC"
	}
	switch req.SortBy {
	case "title":
		orderClause = "papers.title " + dir
	case "year", "publication_year":
		orderClause = "papers.publication_year " + dir
	case "updated_at":
		orderClause = "papers.updated_at " + dir
	default:
		orderClause = "papers.created_at " + dir
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var papers []model.Paper
	if err := q.Order(orderClause).Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&papers).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}

	list := make([]*protopaper.PaperItem, 0, len(papers))
	for i := range papers {
		list = append(list, s.enrichPaperItem(ctx, &papers[i]))
	}

	return &protopaper.ListPapersResp{
		List:  list,
		Total: total,
	}, nil
}

func (s *PaperServer) GetPaperDetail(ctx context.Context, req *protosdkws.IDReq) (*protopaper.GetPaperDetailResp, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var p model.Paper
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&p).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("论文不存在或无访问权限")
	}

	var secs []model.PaperSection
	s.db.WithContext(ctx).Where("paper_id = ?", p.ID).Order("section_order asc").Find(&secs)
	protoSecs := make([]*protopaper.PaperSectionItem, 0, len(secs))
	for _, sec := range secs {
		protoSecs = append(protoSecs, &protopaper.PaperSectionItem{
			Id:           sec.ID,
			SectionTitle: sec.SectionTitle,
			SectionType:  sec.SectionType,
			StartPage:    sec.StartPage,
			EndPage:      sec.EndPage,
			Content:      sec.Content,
			SectionOrder: sec.SectionOrder,
		})
	}

	analysisData, _ := s.getAnalysisData(ctx, p.ID)

	var job model.ProcessingJob
	s.db.WithContext(ctx).Where("paper_id = ?", p.ID).Order("id desc").First(&job)

	return &protopaper.GetPaperDetailResp{
		Paper:     s.enrichPaperItem(ctx, &p),
		Sections:  protoSecs,
		Analysis:  analysisData,
		LatestJob: toProtoJob(&job),
	}, nil
}

func (s *PaperServer) getAnalysisData(ctx context.Context, paperID int64) (*protopaper.PaperAnalysisData, error) {
	var pa model.PaperAnalysis
	if err := s.db.WithContext(ctx).Where("paper_id = ?", paperID).First(&pa).Error; err != nil {
		emptyField := &protopaper.AnalysisField{Value: "尚未提取", SourceType: "unverified", VerificationStatus: "unverified"}
		return &protopaper.PaperAnalysisData{
			PaperId:           paperID,
			ResearchProblem:   emptyField,
			ResearchObjective: emptyField,
			Contributions:     emptyField,
			Methodology:       emptyField,
			Datasets:          emptyField,
			ExpSettings:       emptyField,
			Metrics:           emptyField,
			Results:           emptyField,
			Limitations:       emptyField,
			FutureWork:        emptyField,
			AnalysisVersion:   0,
		}, nil
	}

	return &protopaper.PaperAnalysisData{
		Id:                pa.ID,
		PaperId:           pa.PaperID,
		ResearchProblem:   unmarshalAnalysisField(pa.ResearchProblem),
		ResearchObjective: unmarshalAnalysisField(pa.ResearchObjective),
		Contributions:     unmarshalAnalysisField(pa.Contributions),
		Methodology:       unmarshalAnalysisField(pa.Methodology),
		Datasets:          unmarshalAnalysisField(pa.Datasets),
		ExpSettings:       unmarshalAnalysisField(pa.ExpSettings),
		Metrics:           unmarshalAnalysisField(pa.Metrics),
		Results:           unmarshalAnalysisField(pa.Results),
		Limitations:       unmarshalAnalysisField(pa.Limitations),
		FutureWork:        unmarshalAnalysisField(pa.FutureWork),
		ExtraJson:         pa.ExtraJSON,
		AnalysisVersion:   pa.AnalysisVersion,
		UpdatedAt:         pa.UpdatedAt.Unix(),
	}, nil
}

func (s *PaperServer) UpdatePaperMetadata(ctx context.Context, req *protopaper.UpdatePaperMetadataReq) (*protopaper.PaperItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var p model.Paper
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&p).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("论文不存在")
	}

	if strings.TrimSpace(req.Title) != "" {
		p.Title = strings.TrimSpace(req.Title)
	}
	p.Abstract = req.Abstract
	if req.Authors != nil {
		p.Authors = marshalStringSlice(req.Authors)
	}
	if req.PublicationYear > 0 {
		p.PublicationYear = req.PublicationYear
	}
	p.Journal = req.Journal
	p.DOI = req.Doi
	if req.Language != "" {
		p.Language = req.Language
	}
	if req.PaperType != "" {
		p.PaperType = req.PaperType
	}
	if req.Keywords != nil {
		p.Keywords = marshalStringSlice(req.Keywords)
	}
	p.PublicationURL = req.PublicationUrl

	if err := s.db.WithContext(ctx).Save(&p).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}

	return s.enrichPaperItem(ctx, &p), nil
}

func (s *PaperServer) DeletePaper(ctx context.Context, req *protosdkws.IDReq) (*protosdkws.Empty, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).Delete(&model.Paper{}).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}
	return &protosdkws.Empty{}, nil
}

func (s *PaperServer) GetPaperFilePath(ctx context.Context, req *protosdkws.IDReq) (*protopaper.GetFilePathResp, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var p model.Paper
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&p).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("无权访问该论文文件")
	}
	var pf model.PaperFile
	if err := s.db.WithContext(ctx).Where("paper_id = ?", p.ID).Order("id desc").First(&pf).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("PDF 文件不存在")
	}
	return &protopaper.GetFilePathResp{
		FilePath:         pf.FilePath,
		OriginalFilename: pf.OriginalFilename,
	}, nil
}

func (s *PaperServer) GetPaperPages(ctx context.Context, req *protosdkws.IDReq) (*protopaper.GetPaperPagesResp, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var p model.Paper
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&p).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("论文不存在")
	}
	var pages []model.PaperPage
	s.db.WithContext(ctx).Where("paper_id = ?", p.ID).Order("page_number asc").Find(&pages)
	var secs []model.PaperSection
	s.db.WithContext(ctx).Where("paper_id = ?", p.ID).Order("section_order asc").Find(&secs)

	resp := &protopaper.GetPaperPagesResp{}
	for _, pg := range pages {
		resp.Pages = append(resp.Pages, &protopaper.PaperPageItem{
			PageNumber: pg.PageNumber,
			RawText:    pg.RawText,
			CleanText:  pg.CleanText,
		})
	}
	for _, sec := range secs {
		resp.Sections = append(resp.Sections, &protopaper.PaperSectionItem{
			Id:           sec.ID,
			SectionTitle: sec.SectionTitle,
			SectionType:  sec.SectionType,
			StartPage:    sec.StartPage,
			EndPage:      sec.EndPage,
			Content:      sec.Content,
			SectionOrder: sec.SectionOrder,
		})
	}
	return resp, nil
}

func (s *PaperServer) TriggerParseAndAnalyze(ctx context.Context, req *protosdkws.IDReq) (*protopaper.ProcessingJobItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var p model.Paper
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&p).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("论文不存在")
	}
	return s.startParseJob(uid, p.ID, true)
}

func (s *PaperServer) GetJobStatus(ctx context.Context, req *protosdkws.IDReq) (*protopaper.ProcessingJobItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var job model.ProcessingJob
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&job).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("任务不存在")
	}
	return toProtoJob(&job), nil
}

func (s *PaperServer) GetPaperAnalysis(ctx context.Context, req *protosdkws.IDReq) (*protopaper.PaperAnalysisData, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var p model.Paper
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&p).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("论文不存在")
	}
	return s.getAnalysisData(ctx, p.ID)
}

func (s *PaperServer) UpdatePaperAnalysis(ctx context.Context, req *protopaper.UpdatePaperAnalysisReq) (*protopaper.PaperAnalysisData, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var p model.Paper
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.PaperId, uid).First(&p).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("论文不存在")
	}
	if req.Analysis == nil {
		return nil, errs.ErrArgs.WrapMsg("analysis payload is nil")
	}

	var pa model.PaperAnalysis
	err = s.db.WithContext(ctx).Where("paper_id = ?", p.ID).First(&pa).Error
	if err != nil {
		pa.ID, _ = s.idGen.NextID()
		pa.PaperID = p.ID
		pa.AnalysisVersion = 1
	} else {
		pa.AnalysisVersion++
	}

	pa.ResearchProblem = marshalAnalysisField(req.Analysis.ResearchProblem)
	pa.ResearchObjective = marshalAnalysisField(req.Analysis.ResearchObjective)
	pa.Contributions = marshalAnalysisField(req.Analysis.Contributions)
	pa.Methodology = marshalAnalysisField(req.Analysis.Methodology)
	pa.Datasets = marshalAnalysisField(req.Analysis.Datasets)
	pa.ExpSettings = marshalAnalysisField(req.Analysis.ExpSettings)
	pa.Metrics = marshalAnalysisField(req.Analysis.Metrics)
	pa.Results = marshalAnalysisField(req.Analysis.Results)
	pa.Limitations = marshalAnalysisField(req.Analysis.Limitations)
	pa.FutureWork = marshalAnalysisField(req.Analysis.FutureWork)
	pa.ExtraJSON = req.Analysis.ExtraJson
	pa.UpdatedAt = time.Now()

	if err != nil {
		s.db.WithContext(ctx).Create(&pa)
	} else {
		s.db.WithContext(ctx).Save(&pa)
	}
	s.db.WithContext(ctx).Model(&model.Paper{}).Where("id = ?", p.ID).Update("parse_status", "completed")
	return s.getAnalysisData(ctx, p.ID)
}

func (s *PaperServer) GetDashboardOverview(ctx context.Context, _ *protosdkws.Empty) (*protopaper.DashboardOverviewResp, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	resp := &protopaper.DashboardOverviewResp{}
	s.db.WithContext(ctx).Model(&model.Paper{}).Where("user_id = ?", uid).Count(&resp.TotalPapers)
	s.db.WithContext(ctx).Model(&model.Comparison{}).Where("user_id = ?", uid).Count(&resp.TotalComparisons)

	// Recent added papers
	var recentPapers []model.Paper
	s.db.WithContext(ctx).Where("user_id = ?", uid).Order("created_at desc").Limit(5).Find(&recentPapers)
	for i := range recentPapers {
		resp.RecentAdded = append(resp.RecentAdded, s.enrichPaperItem(ctx, &recentPapers[i]))
	}

	return resp, nil
}

func computeFileSHA256(path string) (string, int64) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()
	h := sha256.New()
	sz, _ := io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), sz
}
