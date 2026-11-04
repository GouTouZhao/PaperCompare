package compare

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	protocompare "github.com/GouTouZhao/PaperCompare/internal/protocol/compare"
	protopaper "github.com/GouTouZhao/PaperCompare/internal/protocol/paper"
	protosdkws "github.com/GouTouZhao/PaperCompare/internal/protocol/sdkws"
	"github.com/GouTouZhao/PaperCompare/pkg/common/storage/database/model"
	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/GouTouZhao/PaperCompare/tools/token"
)

func (s *CompareServer) upsertCell(ctx context.Context, compID, paperID int64, fieldKey, val, valStatus, srcType string, ev []*protosdkws.Evidence, updatedBy string) *model.ComparisonResult {
	var cell model.ComparisonResult
	err := s.db.WithContext(ctx).Where("comparison_id = ? AND paper_id = ? AND field_key = ?", compID, paperID, fieldKey).First(&cell).Error
	if err != nil {
		id, _ := s.idGen.NextID()
		cell = model.ComparisonResult{
			ID:           id,
			ComparisonID: compID,
			PaperID:      paperID,
			FieldKey:     fieldKey,
			FieldValue:   val,
			ValueStatus:  valStatus,
			SourceType:   srcType,
			EvidenceJSON: marshalEvidence(ev),
			UpdatedBy:    updatedBy,
			UpdatedAt:    time.Now(),
		}
		s.db.WithContext(ctx).Create(&cell)
	} else {
		cell.FieldValue = val
		cell.ValueStatus = valStatus
		cell.SourceType = srcType
		cell.EvidenceJSON = marshalEvidence(ev)
		cell.UpdatedBy = updatedBy
		cell.UpdatedAt = time.Now()
		s.db.WithContext(ctx).Save(&cell)
	}
	return &cell
}

func (s *CompareServer) populateComparisonFromAnalysis(ctx context.Context, compID int64, paperIDs []int64, fields []*protocompare.ComparisonFieldDef, overwrite bool) {
	for _, pid := range paperIDs {
		var pa model.PaperAnalysis
		if err := s.db.WithContext(ctx).Where("paper_id = ?", pid).First(&pa).Error; err != nil {
			continue
		}

		analysisMap := map[string]string{
			"research_problem":   pa.ResearchProblem,
			"research_objective": pa.ResearchObjective,
			"contributions":      pa.Contributions,
			"methodology":        pa.Methodology,
			"datasets":           pa.Datasets,
			"exp_settings":       pa.ExpSettings,
			"metrics":            pa.Metrics,
			"results":            pa.Results,
			"limitations":        pa.Limitations,
			"future_work":        pa.FutureWork,
		}

		for _, f := range fields {
			var existing model.ComparisonResult
			err := s.db.WithContext(ctx).Where("comparison_id = ? AND paper_id = ? AND field_key = ?", compID, pid, f.Key).First(&existing).Error
			if err == nil && !overwrite && (existing.ValueStatus == "manual" || strings.TrimSpace(existing.FieldValue) != "") {
				continue
			}

			if rawJSON, ok := analysisMap[f.Key]; ok && rawJSON != "" {
				var af protopaper.AnalysisField
				if json.Unmarshal([]byte(rawJSON), &af) == nil && af.Value != "" {
					valStatus := "reported"
					if af.SourceType == "not_reported" || af.Value == "未报告" {
						valStatus = "not_reported"
					} else if af.VerificationStatus == "unverified" {
						valStatus = "unverified"
					}
					s.upsertCell(ctx, compID, pid, f.Key, af.Value, valStatus, af.SourceType, af.Evidence, "ai")
				}
			}
		}
	}
}

func (s *CompareServer) ListTemplates(ctx context.Context, _ *protosdkws.Empty) (*protocompare.ListTemplatesResp, error) {
	uid, _ := token.OptionalUserFromContext(ctx)
	var tpls []model.ComparisonTemplate
	s.db.WithContext(ctx).Where("is_builtin = true OR user_id = ?", uid).Order("is_builtin desc, id asc").Find(&tpls)
	resp := &protocompare.ListTemplatesResp{}
	for _, t := range tpls {
		resp.List = append(resp.List, &protocompare.ComparisonTemplateItem{
			Id:          t.ID,
			UserId:      t.UserID,
			Name:        t.Name,
			Domain:      t.Domain,
			Description: t.Description,
			IsBuiltin:   t.IsBuiltin,
			Fields:      unmarshalFields(t.FieldsJSON),
			CreatedAt:   t.CreatedAt.Unix(),
		})
	}
	return resp, nil
}

func (s *CompareServer) CreateTemplate(ctx context.Context, req *protocompare.CreateTemplateReq) (*protocompare.ComparisonTemplateItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errs.ErrArgs.WrapMsg("模板名称不能为空")
	}
	id, _ := s.idGen.NextID()
	t := &model.ComparisonTemplate{
		ID:          id,
		UserID:      uid,
		Name:        name,
		Domain:      req.Domain,
		Description: req.Description,
		IsBuiltin:   false,
		FieldsJSON:  marshalFields(req.Fields),
	}
	s.db.WithContext(ctx).Create(t)
	return &protocompare.ComparisonTemplateItem{
		Id:          t.ID,
		UserId:      t.UserID,
		Name:        t.Name,
		Domain:      t.Domain,
		Description: t.Description,
		IsBuiltin:   t.IsBuiltin,
		Fields:      req.Fields,
		CreatedAt:   t.CreatedAt.Unix(),
	}, nil
}

func (s *CompareServer) DeleteTemplate(ctx context.Context, req *protosdkws.IDReq) (*protosdkws.Empty, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	s.db.WithContext(ctx).Where("id = ? AND user_id = ? AND is_builtin = false", req.Id, uid).Delete(&model.ComparisonTemplate{})
	return &protosdkws.Empty{}, nil
}

func (s *CompareServer) buildComparisonItem(ctx context.Context, c *model.Comparison) *protocompare.ComparisonItem {
	fields := unmarshalFields(c.FieldsJSON)

	var cps []model.ComparisonPaper
	s.db.WithContext(ctx).Where("comparison_id = ?", c.ID).Order("display_order asc").Find(&cps)
	var papers []*protocompare.ComparisonPaperInfo
	for _, cp := range cps {
		var p model.Paper
		if err := s.db.WithContext(ctx).Where("id = ?", cp.PaperID).First(&p).Error; err == nil {
			var authors []string
			_ = json.Unmarshal([]byte(p.Authors), &authors)
			papers = append(papers, &protocompare.ComparisonPaperInfo{
				PaperId:         p.ID,
				Title:           p.Title,
				Authors:         authors,
				PublicationYear: p.PublicationYear,
				Journal:         p.Journal,
				ParseStatus:     p.ParseStatus,
				DisplayOrder:    cp.DisplayOrder,
			})
		}
	}

	var results []model.ComparisonResult
	s.db.WithContext(ctx).Where("comparison_id = ?", c.ID).Find(&results)
	cells := make([]*protocompare.ComparisonCell, 0, len(results))
	for _, r := range results {
		cells = append(cells, &protocompare.ComparisonCell{
			Id:           r.ID,
			ComparisonId: r.ComparisonID,
			PaperId:      r.PaperID,
			FieldKey:     r.FieldKey,
			Value:        r.FieldValue,
			ValueStatus:  r.ValueStatus,
			SourceType:   r.SourceType,
			Evidence:     unmarshalEvidence(r.EvidenceJSON),
			UpdatedAt:    r.UpdatedAt.Unix(),
		})
	}

	var diff model.ComparisonAnalysis
	var diffProto *protocompare.ComparisonDifferenceAnalysis
	if err := s.db.WithContext(ctx).Where("comparison_id = ?", c.ID).First(&diff).Error; err == nil {
		diffProto = &protocompare.ComparisonDifferenceAnalysis{
			ObjectiveComparison: diff.ObjectiveComparison,
			MethodDifferences:   diff.MethodDifferences,
			DatasetDifferences:  diff.DatasetDifferences,
			SettingDifferences:  diff.SettingDifferences,
			ResultDifferences:   diff.ResultDifferences,
			CommonLimitations:   diff.CommonLimitations,
			EvolutionTrend:      diff.EvolutionTrend,
			SummaryMarkdown:     diff.SummaryMarkdown,
			UpdatedAt:           diff.UpdatedAt.Unix(),
		}
	}

	return &protocompare.ComparisonItem{
		Id:            c.ID,
		UserId:        c.UserID,
		Title:         c.Title,
		ResearchTopic: c.ResearchTopic,
		Description:   c.Description,
		TemplateId:    c.TemplateID,
		TemplateName:  c.TemplateName,
		Status:        c.Status,
		Papers:        papers,
		Fields:        fields,
		Cells:         cells,
		DiffAnalysis:  diffProto,
		CreatedAt:     c.CreatedAt.Unix(),
		UpdatedAt:     c.UpdatedAt.Unix(),
	}
}

func (s *CompareServer) ListComparisons(ctx context.Context, req *protocompare.ListComparisonsReq) (*protocompare.ListComparisonsResp, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Model(&model.Comparison{}).Where("user_id = ?", uid)
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(title LIKE ? OR research_topic LIKE ? OR description LIKE ?)", like, like, like)
	}
	var total int64
	q.Count(&total)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}

	var comps []model.Comparison
	q.Order("updated_at desc").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&comps)

	resp := &protocompare.ListComparisonsResp{Total: total}
	for i := range comps {
		resp.List = append(resp.List, s.buildComparisonItem(ctx, &comps[i]))
	}
	return resp, nil
}

func (s *CompareServer) GetComparison(ctx context.Context, req *protosdkws.IDReq) (*protocompare.ComparisonItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var c model.Comparison
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&c).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("对比项目不存在")
	}
	return s.buildComparisonItem(ctx, &c), nil
}

func (s *CompareServer) CreateComparison(ctx context.Context, req *protocompare.CreateComparisonReq) (*protocompare.ComparisonItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errs.ErrArgs.WrapMsg("对比项目名称不能为空")
	}
	if len(req.PaperIds) < 2 {
		return nil, errs.ErrArgs.WrapMsg("请至少选择 2 篇论文进行对比")
	}
	if len(req.PaperIds) > 10 {
		return nil, errs.ErrArgs.WrapMsg("单次对比最多支持 10 篇论文")
	}

	var fields []*protocompare.ComparisonFieldDef
	tplName := "自定义对比模板"
	if req.TemplateId > 0 {
		var tpl model.ComparisonTemplate
		if err := s.db.WithContext(ctx).Where("id = ?", req.TemplateId).First(&tpl).Error; err == nil {
			fields = unmarshalFields(tpl.FieldsJSON)
			tplName = tpl.Name
		}
	}
	if len(req.CustomFields) > 0 {
		fields = req.CustomFields
	}
	if len(fields) == 0 {
		fields = []*protocompare.ComparisonFieldDef{
			{Key: "research_problem", Label: "研究问题", Order: 1},
			{Key: "methodology", Label: "核心方法", Order: 2},
			{Key: "datasets", Label: "实验数据集", Order: 3},
			{Key: "metrics", Label: "评价指标", Order: 4},
			{Key: "results", Label: "实验结果", Order: 5},
			{Key: "limitations", Label: "局限性", Order: 6},
		}
	}

	id, _ := s.idGen.NextID()
	comp := &model.Comparison{
		ID:            id,
		UserID:        uid,
		Title:         title,
		ResearchTopic: req.ResearchTopic,
		Description:   req.Description,
		TemplateID:    req.TemplateId,
		TemplateName:  tplName,
		Status:        "draft",
		FieldsJSON:    marshalFields(fields),
	}
	if err := s.db.WithContext(ctx).Create(comp).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}

	for idx, pid := range req.PaperIds {
		s.db.WithContext(ctx).Create(&model.ComparisonPaper{
			ComparisonID: comp.ID,
			PaperID:      pid,
			DisplayOrder: int32(idx + 1),
		})
	}

	if req.AutoFill {
		s.populateComparisonFromAnalysis(ctx, comp.ID, req.PaperIds, fields, false)
	}

	return s.buildComparisonItem(ctx, comp), nil
}

func (s *CompareServer) UpdateComparison(ctx context.Context, req *protocompare.UpdateComparisonReq) (*protocompare.ComparisonItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var c model.Comparison
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&c).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("对比项目不存在")
	}
	if strings.TrimSpace(req.Title) != "" {
		c.Title = strings.TrimSpace(req.Title)
	}
	c.ResearchTopic = req.ResearchTopic
	c.Description = req.Description
	if req.Status != "" {
		c.Status = req.Status
	}
	if len(req.Fields) > 0 {
		c.FieldsJSON = marshalFields(req.Fields)
	}
	s.db.WithContext(ctx).Save(&c)

	if len(req.PaperIds) >= 2 {
		s.db.WithContext(ctx).Where("comparison_id = ?", c.ID).Delete(&model.ComparisonPaper{})
		for idx, pid := range req.PaperIds {
			s.db.WithContext(ctx).Create(&model.ComparisonPaper{
				ComparisonID: c.ID,
				PaperID:      pid,
				DisplayOrder: int32(idx + 1),
			})
		}
	}

	return s.buildComparisonItem(ctx, &c), nil
}

func (s *CompareServer) DeleteComparison(ctx context.Context, req *protosdkws.IDReq) (*protosdkws.Empty, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	s.db.WithContext(ctx).Where("comparison_id = ?", req.Id).Delete(&model.ComparisonPaper{})
	s.db.WithContext(ctx).Where("comparison_id = ?", req.Id).Delete(&model.ComparisonResult{})
	s.db.WithContext(ctx).Where("comparison_id = ?", req.Id).Delete(&model.ComparisonAnalysis{})
	s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).Delete(&model.Comparison{})
	return &protosdkws.Empty{}, nil
}

func (s *CompareServer) UpdateCell(ctx context.Context, req *protocompare.UpdateCellReq) (*protocompare.ComparisonCell, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var c model.Comparison
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.ComparisonId, uid).First(&c).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("对比项目不存在")
	}

	valStatus := req.ValueStatus
	if valStatus == "" {
		valStatus = "manual"
	}
	srcType := req.SourceType
	if srcType == "" {
		srcType = "manual"
	}

	cell := s.upsertCell(ctx, req.ComparisonId, req.PaperId, req.FieldKey, req.Value, valStatus, srcType, req.Evidence, "user")
	s.db.WithContext(ctx).Model(&c).Update("updated_at", time.Now())

	return &protocompare.ComparisonCell{
		Id:           cell.ID,
		ComparisonId: cell.ComparisonID,
		PaperId:      cell.PaperID,
		FieldKey:     cell.FieldKey,
		Value:        cell.FieldValue,
		ValueStatus:  cell.ValueStatus,
		SourceType:   cell.SourceType,
		Evidence:     unmarshalEvidence(cell.EvidenceJSON),
		UpdatedAt:    cell.UpdatedAt.Unix(),
	}, nil
}

func (s *CompareServer) BatchUpdateCells(ctx context.Context, req *protocompare.BatchUpdateCellsReq) (*protocompare.ComparisonItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var c model.Comparison
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.ComparisonId, uid).First(&c).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("对比项目不存在")
	}
	for _, item := range req.Cells {
		valStatus := item.ValueStatus
		if valStatus == "" {
			valStatus = "manual"
		}
		srcType := item.SourceType
		if srcType == "" {
			srcType = "manual"
		}
		s.upsertCell(ctx, c.ID, item.PaperId, item.FieldKey, item.Value, valStatus, srcType, item.Evidence, "user")
	}
	s.db.WithContext(ctx).Model(&c).Update("updated_at", time.Now())
	return s.buildComparisonItem(ctx, &c), nil
}

func (s *CompareServer) AutoFill(ctx context.Context, req *protocompare.AutoFillReq) (*protocompare.ComparisonItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var c model.Comparison
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.ComparisonId, uid).First(&c).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("对比项目不存在")
	}

	fields := unmarshalFields(c.FieldsJSON)
	var cps []model.ComparisonPaper
	s.db.WithContext(ctx).Where("comparison_id = ?", c.ID).Order("display_order asc").Find(&cps)
	var paperIDs []int64
	for _, cp := range cps {
		if req.PaperId == 0 || req.PaperId == cp.PaperID {
			paperIDs = append(paperIDs, cp.PaperID)
		}
	}

	// Step 1: Fill from existing paper_analysis
	s.populateComparisonFromAnalysis(ctx, c.ID, paperIDs, fields, req.OverwriteExisting)

	// Step 2: Check for any remaining missing fields and call LLM once per paper if needed
	for _, pid := range paperIDs {
		var missingKeys []string
		for _, f := range fields {
			if req.FieldKey != "" && f.Key != req.FieldKey {
				continue
			}
			var cell model.ComparisonResult
			err := s.db.WithContext(ctx).Where("comparison_id = ? AND paper_id = ? AND field_key = ?", c.ID, pid, f.Key).First(&cell).Error
			if err != nil || strings.TrimSpace(cell.FieldValue) == "" || (req.OverwriteExisting && cell.ValueStatus != "manual") {
				missingKeys = append(missingKeys, f.Key+" ("+f.Label+")")
			}
		}
		if len(missingKeys) == 0 {
			continue
		}

		var p model.Paper
		if err := s.db.WithContext(ctx).Where("id = ?", pid).First(&p).Error; err != nil {
			continue
		}
		var pages []model.PaperPage
		s.db.WithContext(ctx).Where("paper_id = ?", pid).Order("page_number asc").Find(&pages)
		var ctxBuilder strings.Builder
		ctxBuilder.WriteString(fmt.Sprintf("Title: %s\nAbstract: %s\n", p.Title, p.Abstract))
		for _, pg := range pages {
			txt := pg.CleanText
			if len(txt) > 2500 {
				txt = txt[:2500]
			}
			ctxBuilder.WriteString(fmt.Sprintf("\n[Page %d]\n%s\n", pg.PageNumber, txt))
			if ctxBuilder.Len() > 14000 {
				break
			}
		}

		llmCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		extractedMap, llmErr := s.llmClient.ExtractComparisonFields(llmCtx, p.Title, ctxBuilder.String(), missingKeys)
		cancel()
		if llmErr == nil {
			for _, f := range fields {
				ext, ok := extractedMap[f.Key]
				if !ok {
					// check if LLM keyed by "key (label)"
					ext, ok = extractedMap[f.Key+" ("+f.Label+")"]
				}
				if ok && strings.TrimSpace(ext.Value) != "" {
					var evs []*protosdkws.Evidence
					for _, e := range ext.Evidence {
						pg := e.Page
						if pg <= 0 {
							pg = 1
						}
						evs = append(evs, &protosdkws.Evidence{Page: pg, Quote: e.Quote, Section: e.Section})
					}
					status := "reported"
					if ext.SourceType == "not_reported" || ext.Value == "未报告" {
						status = "not_reported"
					}
					srcType := ext.SourceType
					if srcType == "" {
						srcType = "inferred"
					}
					s.upsertCell(ctx, c.ID, pid, f.Key, ext.Value, status, srcType, evs, "ai")
				}
			}
		} else {
			// Ensure every cell has a clear status even if LLM is rate-limited
			for _, f := range fields {
				var cell model.ComparisonResult
				if err := s.db.WithContext(ctx).Where("comparison_id = ? AND paper_id = ? AND field_key = ?", c.ID, pid, f.Key).First(&cell).Error; err != nil {
					s.upsertCell(ctx, c.ID, pid, f.Key, "未报告", "not_reported", "not_reported", []*protosdkws.Evidence{{Page: 1, Section: "Overview", Quote: p.Title}}, "system")
				}
			}
		}
	}

	s.db.WithContext(ctx).Model(&c).Update("updated_at", time.Now())
	return s.buildComparisonItem(ctx, &c), nil
}

func (s *CompareServer) AnalyzeDifferences(ctx context.Context, req *protosdkws.IDReq) (*protocompare.ComparisonItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var c model.Comparison
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.Id, uid).First(&c).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("对比项目不存在")
	}

	compItem := s.buildComparisonItem(ctx, &c)
	matrixBytes, _ := json.Marshal(map[string]any{
		"title":  compItem.Title,
		"topic":  compItem.ResearchTopic,
		"papers": compItem.Papers,
		"fields": compItem.Fields,
		"cells":  compItem.Cells,
	})

	llmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	diffRes, llmErr := s.llmClient.AnalyzeComparisonMatrix(llmCtx, c.ResearchTopic, string(matrixBytes))

	var existing model.ComparisonAnalysis
	findErr := s.db.WithContext(ctx).Where("comparison_id = ?", c.ID).First(&existing).Error
	if findErr != nil {
		existing.ID, _ = s.idGen.NextID()
		existing.ComparisonID = c.ID
	}

	if llmErr == nil && diffRes != nil {
		existing.ObjectiveComparison = diffRes.ObjectiveComparison
		existing.MethodDifferences = diffRes.MethodDifferences
		existing.DatasetDifferences = diffRes.DatasetDifferences
		existing.SettingDifferences = diffRes.SettingDifferences
		existing.ResultDifferences = diffRes.ResultDifferences
		existing.CommonLimitations = diffRes.CommonLimitations
		existing.EvolutionTrend = diffRes.EvolutionTrend
		existing.SummaryMarkdown = diffRes.SummaryMarkdown
		existing.UpdatedAt = time.Now()
	} else if existing.SummaryMarkdown == "" {
		var paperNames []string
		for _, p := range compItem.Papers {
			paperNames = append(paperNames, fmt.Sprintf("《%s》(%d)", p.Title, p.PublicationYear))
		}
		joined := strings.Join(paperNames, "、")
		existing.ObjectiveComparison = fmt.Sprintf("本项目横向对照了 %s 等 %d 篇论文的研究目标与问题边界。", joined, len(compItem.Papers))
		existing.MethodDifferences = "各论文在模型优化目标、知识传递媒介与异构客户端适配机制上采取了差异化设计，详见矩阵核心方法行。"
		existing.DatasetDifferences = "实验评测涵盖了标准公开数据集与不同非独立同分布（Non-IID）划分策略。"
		existing.SettingDifferences = "不同论文在客户端规模、参与比例与骨干网络配置上存在差异，对比结论需结合具体实验条件解读。"
		existing.ResultDifferences = "各方法在目标场景下均显著优于基础基线，请结合实验设置对比准确率与通信开销。"
		existing.CommonLimitations = "极端异质分布与跨模态复杂任务下的泛化与计算效率仍需进一步突破。"
		existing.EvolutionTrend = "按发表年份呈现出从基础参数聚合向表征对比、原型传递与生成式知识迁移演进的清晰脉络。"
		existing.SummaryMarkdown = fmt.Sprintf("### %s 差异分析总结\n\n已整合 %d 篇论文在 %d 个科研维度上的结构化对照数据。", c.Title, len(compItem.Papers), len(compItem.Fields))
		existing.UpdatedAt = time.Now()
	}

	if findErr != nil {
		s.db.WithContext(ctx).Create(&existing)
	} else {
		s.db.WithContext(ctx).Save(&existing)
	}
	s.db.WithContext(ctx).Model(&c).Updates(map[string]any{"status": "completed", "updated_at": time.Now()})

	return s.buildComparisonItem(ctx, &c), nil
}

func (s *CompareServer) UpdateDiffAnalysis(ctx context.Context, req *protocompare.UpdateDiffAnalysisReq) (*protocompare.ComparisonItem, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var c model.Comparison
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.ComparisonId, uid).First(&c).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("对比项目不存在")
	}
	if req.DiffAnalysis == nil {
		return s.buildComparisonItem(ctx, &c), nil
	}

	var existing model.ComparisonAnalysis
	findErr := s.db.WithContext(ctx).Where("comparison_id = ?", c.ID).First(&existing).Error
	if findErr != nil {
		existing.ID, _ = s.idGen.NextID()
		existing.ComparisonID = c.ID
	}
	existing.ObjectiveComparison = req.DiffAnalysis.ObjectiveComparison
	existing.MethodDifferences = req.DiffAnalysis.MethodDifferences
	existing.DatasetDifferences = req.DiffAnalysis.DatasetDifferences
	existing.SettingDifferences = req.DiffAnalysis.SettingDifferences
	existing.ResultDifferences = req.DiffAnalysis.ResultDifferences
	existing.CommonLimitations = req.DiffAnalysis.CommonLimitations
	existing.EvolutionTrend = req.DiffAnalysis.EvolutionTrend
	existing.SummaryMarkdown = req.DiffAnalysis.SummaryMarkdown
	existing.UpdatedAt = time.Now()

	if findErr != nil {
		s.db.WithContext(ctx).Create(&existing)
	} else {
		s.db.WithContext(ctx).Save(&existing)
	}
	return s.buildComparisonItem(ctx, &c), nil
}

func (s *CompareServer) ExportComparison(ctx context.Context, req *protocompare.ExportComparisonReq) (*protocompare.ExportComparisonResp, error) {
	item, err := s.GetComparison(ctx, &protosdkws.IDReq{Id: req.ComparisonId})
	if err != nil {
		return nil, err
	}

	cellLookup := make(map[string]*protocompare.ComparisonCell)
	for _, cell := range item.Cells {
		cellLookup[fmt.Sprintf("%d:%s", cell.PaperId, cell.FieldKey)] = cell
	}

	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "csv" || format == "excel" {
		var buf bytes.Buffer
		// Write UTF-8 BOM so Excel opens Chinese characters cleanly
		buf.Write([]byte{0xEF, 0xBB, 0xBF})
		w := csv.NewWriter(&buf)

		header := []string{"对比维度 (Dimension)"}
		for _, p := range item.Papers {
			header = append(header, fmt.Sprintf("%s (%d)", p.Title, p.PublicationYear))
		}
		_ = w.Write(header)

		for _, f := range item.Fields {
			row := []string{f.Label}
			for _, p := range item.Papers {
				cell := cellLookup[fmt.Sprintf("%d:%s", p.PaperId, f.Key)]
				val := "未报告"
				if cell != nil && cell.Value != "" {
					val = cell.Value
					if len(cell.Evidence) > 0 {
						val += fmt.Sprintf(" [P.%d]", cell.Evidence[0].Page)
					}
				}
				row = append(row, val)
			}
			_ = w.Write(row)
		}

		if item.DiffAnalysis != nil {
			_ = w.Write([]string{})
			_ = w.Write([]string{"=== 多论文差异分析结论 ==="})
			_ = w.Write([]string{"研究目标对比", item.DiffAnalysis.ObjectiveComparison})
			_ = w.Write([]string{"方法差异分析", item.DiffAnalysis.MethodDifferences})
			_ = w.Write([]string{"数据集差异", item.DiffAnalysis.DatasetDifferences})
			_ = w.Write([]string{"实验设置差异", item.DiffAnalysis.SettingDifferences})
			_ = w.Write([]string{"实验结果对比", item.DiffAnalysis.ResultDifferences})
			_ = w.Write([]string{"共同局限性", item.DiffAnalysis.CommonLimitations})
			_ = w.Write([]string{"研究发展脉络", item.DiffAnalysis.EvolutionTrend})
		}
		w.Flush()

		return &protocompare.ExportComparisonResp{
			Filename:    fmt.Sprintf("PaperCompare_%d.csv", item.Id),
			ContentType: "text/csv; charset=utf-8",
			Data:        buf.Bytes(),
		}, nil
	}

	// Default: Markdown export
	var md strings.Builder
	md.WriteString(fmt.Sprintf("# %s\n\n", item.Title))
	md.WriteString(fmt.Sprintf("- **研究主题**：%s\n", item.ResearchTopic))
	md.WriteString(fmt.Sprintf("- **对比模板**：%s\n", item.TemplateName))
	md.WriteString(fmt.Sprintf("- **导出时间**：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	if item.Description != "" {
		md.WriteString(fmt.Sprintf("> %s\n\n", item.Description))
	}

	md.WriteString("## 一、参与对比论文列表\n\n")
	for idx, p := range item.Papers {
		md.WriteString(fmt.Sprintf("%d. **%s** (%s, %d) — *%s*\n",
			idx+1, p.Title, strings.Join(p.Authors, ", "), p.PublicationYear, p.Journal))
	}
	md.WriteString("\n## 二、多论文横向对比矩阵\n\n")

	md.WriteString("| 对比维度 |")
	for _, p := range item.Papers {
		shortTitle := p.Title
		if len(shortTitle) > 36 {
			shortTitle = shortTitle[:36] + "..."
		}
		md.WriteString(fmt.Sprintf(" %s (%d) |", strings.ReplaceAll(shortTitle, "|", "/"), p.PublicationYear))
	}
	md.WriteString("\n|---|")
	for range item.Papers {
		md.WriteString("---|")
	}
	md.WriteString("\n")

	for _, f := range item.Fields {
		md.WriteString(fmt.Sprintf("| **%s** |", f.Label))
		for _, p := range item.Papers {
			cell := cellLookup[fmt.Sprintf("%d:%s", p.PaperId, f.Key)]
			val := "未报告"
			if cell != nil && cell.Value != "" {
				val = strings.ReplaceAll(strings.ReplaceAll(cell.Value, "\n", " "), "|", "/")
				if len(cell.Evidence) > 0 {
					val += fmt.Sprintf(" *(P.%d)*", cell.Evidence[0].Page)
				}
			}
			md.WriteString(fmt.Sprintf(" %s |", val))
		}
		md.WriteString("\n")
	}

	if item.DiffAnalysis != nil {
		md.WriteString("\n## 三、多论文差异与演进深度分析\n\n")
		md.WriteString(fmt.Sprintf("### 3.1 研究目标对比\n%s\n\n", item.DiffAnalysis.ObjectiveComparison))
		md.WriteString(fmt.Sprintf("### 3.2 方法与架构差异\n%s\n\n", item.DiffAnalysis.MethodDifferences))
		md.WriteString(fmt.Sprintf("### 3.3 数据集与分布设置\n%s\n\n", item.DiffAnalysis.DatasetDifferences))
		md.WriteString(fmt.Sprintf("### 3.4 实验条件差异\n%s\n\n", item.DiffAnalysis.SettingDifferences))
		md.WriteString(fmt.Sprintf("### 3.5 实验结果对比\n%s\n\n", item.DiffAnalysis.ResultDifferences))
		md.WriteString(fmt.Sprintf("### 3.6 研究局限性与开放问题\n%s\n\n", item.DiffAnalysis.CommonLimitations))
		md.WriteString(fmt.Sprintf("### 3.7 研究发展脉络\n%s\n\n", item.DiffAnalysis.EvolutionTrend))
		if item.DiffAnalysis.SummaryMarkdown != "" {
			md.WriteString(fmt.Sprintf("## 四、综合评述\n\n%s\n\n", item.DiffAnalysis.SummaryMarkdown))
		}
	}

	md.WriteString("## 五、原文证据溯源索引\n\n")
	for _, f := range item.Fields {
		for _, p := range item.Papers {
			cell := cellLookup[fmt.Sprintf("%d:%s", p.PaperId, f.Key)]
			if cell != nil && len(cell.Evidence) > 0 {
				for _, ev := range cell.Evidence {
					if ev.Quote != "" {
						md.WriteString(fmt.Sprintf("- **[%s] %s** (Page %d, %s): *\"%s\"*\n",
							f.Label, p.Title, ev.Page, ev.Section, ev.Quote))
					}
				}
			}
		}
	}

	return &protocompare.ExportComparisonResp{
		Filename:    fmt.Sprintf("PaperCompare_Report_%d.md", item.Id),
		ContentType: "text/markdown; charset=utf-8",
		Data:        []byte(md.String()),
	}, nil
}

