package compare

import (
	"context"
	"encoding/json"
	"time"

	protocompare "github.com/GouTouZhao/PaperCompare/internal/protocol/compare"
	protosdkws "github.com/GouTouZhao/PaperCompare/internal/protocol/sdkws"
	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/pkg/common/storage/database/model"
	"github.com/GouTouZhao/PaperCompare/pkg/dbbuild"
	"github.com/GouTouZhao/PaperCompare/tools/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/llm"
	"github.com/GouTouZhao/PaperCompare/tools/snowflake"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type Config struct {
	config.ALLConfig
	ConfigPath config.Path
	Index      config.Index
}

type CompareServer struct {
	protocompare.UnimplementedCompareServer
	config    *Config
	db        *gorm.DB
	idGen     *snowflake.Snowflake
	llmClient *llm.Client
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
	llmCli := llm.NewClient(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
	srv := &CompareServer{
		config:    cfg,
		db:        sqlcli,
		idGen:     idGen,
		llmClient: llmCli,
	}
	srv.seedBuiltinTemplatesAndComparison(ctx)
	protocompare.RegisterCompareServer(server, srv)
	return nil
}

func marshalFields(fields []*protocompare.ComparisonFieldDef) string {
	b, _ := json.Marshal(fields)
	return string(b)
}

func unmarshalFields(raw string) []*protocompare.ComparisonFieldDef {
	var out []*protocompare.ComparisonFieldDef
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

func marshalEvidence(ev []*protosdkws.Evidence) string {
	if ev == nil {
		ev = []*protosdkws.Evidence{}
	}
	b, _ := json.Marshal(ev)
	return string(b)
}

func unmarshalEvidence(raw string) []*protosdkws.Evidence {
	var out []*protosdkws.Evidence
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

func (s *CompareServer) seedBuiltinTemplatesAndComparison(ctx context.Context) {
	var cnt int64
	s.db.WithContext(ctx).Model(&model.ComparisonTemplate{}).Where("is_builtin = true").Count(&cnt)
	if cnt == 0 {
		flFields := []*protocompare.ComparisonFieldDef{
			{Key: "research_problem", Label: "研究问题", Description: "论文针对的联邦学习痛点", Order: 1},
			{Key: "model_heterogeneity", Label: "客户端模型架构", Description: "同构或异构骨干网络支持情况", Order: 2},
			{Key: "knowledge_transfer", Label: "知识传递方式", Description: "参数聚合 / 原型传递 / 隐特征生成蒸馏", Order: 3},
			{Key: "methodology", Label: "核心算法与损失函数", Description: "客户端与服务端优化目标", Order: 4},
			{Key: "datasets", Label: "实验数据集与异质划分", Description: "使用的数据集及 Non-IID 划分方式", Order: 5},
			{Key: "exp_settings", Label: "客户端数量与超参数", Description: "客户端规模、轮数、学习率与骨干", Order: 6},
			{Key: "metrics", Label: "评价指标", Description: "Accuracy / 通信开销 / 收敛轮数", Order: 7},
			{Key: "results", Label: "主要实验结果", Description: "关键性能指标与对比提升", Order: 8},
			{Key: "comm_cost", Label: "通信开销", Description: "单轮传输量或通信压缩比", Order: 9},
			{Key: "limitations", Label: "研究局限性", Description: "作者报告或实验体现的不足", Order: 10},
		}

		kdFields := []*protocompare.ComparisonFieldDef{
			{Key: "research_problem", Label: "研究问题", Description: "知识蒸馏面临的挑战", Order: 1},
			{Key: "teacher_student", Label: "Teacher / Student 架构", Description: "教师与学生模型设计", Order: 2},
			{Key: "distill_target", Label: "蒸馏对象与知识表示", Description: "Logits / 中间特征 / 原型 / 生成特征", Order: 3},
			{Key: "methodology", Label: "蒸馏损失与温度参数", Description: "KL 散度、对比损失或 L2 对齐", Order: 4},
			{Key: "datasets", Label: "实验数据集", Description: "验证数据集及是否需要 Proxy 数据", Order: 5},
			{Key: "results", Label: "实验性能结论", Description: "精度提升与压缩率", Order: 6},
			{Key: "limitations", Label: "局限性", Description: "计算开销或适用条件限制", Order: 7},
		}

		cvFields := []*protocompare.ComparisonFieldDef{
			{Key: "research_objective", Label: "视觉任务与目标", Description: "分类 / 检测 / 分割 / 表示学习", Order: 1},
			{Key: "methodology", Label: "Backbone 与核心模块", Description: "CNN / ViT / 对比学习头", Order: 2},
			{Key: "datasets", Label: "数据集与分辨率", Description: "CIFAR / ImageNet 等", Order: 3},
			{Key: "exp_settings", Label: "训练策略与优化器", Description: "Batch Size、优化器与硬件设置", Order: 4},
			{Key: "metrics", Label: "评价指标", Description: "Top-1 Acc / mAP / FLOPs", Order: 5},
			{Key: "results", Label: "实验结果", Description: "定量结果对比", Order: 6},
			{Key: "limitations", Label: "研究局限性", Description: "显存、算力或泛化限制", Order: 7},
		}

		llmFields := []*protocompare.ComparisonFieldDef{
			{Key: "research_problem", Label: "研究问题", Description: "大模型训练或推理痛点", Order: 1},
			{Key: "methodology", Label: "模型架构与训练范式", Description: "预训练 / 微调 / 对齐策略", Order: 2},
			{Key: "datasets", Label: "训练语料与评测基准", Description: "数据集规模与 Benchmark", Order: 3},
			{Key: "exp_settings", Label: "参数量与上下文长度", Description: "模型规模与训练超参", Order: 4},
			{Key: "results", Label: "核心评测表现", Description: "主要基准得分", Order: 5},
			{Key: "limitations", Label: "局限性", Description: "幻觉、算力或长文本瓶颈", Order: 6},
		}

		templates := []model.ComparisonTemplate{
			{ID: 70001, UserID: 0, Name: "联邦学习方法对比模板 (Federated Learning)", Domain: "Federated Learning", Description: "涵盖模型异构性、知识传递媒介、Non-IID 划分、通信开销与局限性的 10 维标准矩阵", IsBuiltin: true, FieldsJSON: marshalFields(flFields)},
			{ID: 70002, UserID: 0, Name: "知识蒸馏对比模板 (Knowledge Distillation)", Domain: "Knowledge Distillation", Description: "面向教师-学生架构、蒸馏特征层级与无数据生成蒸馏的横向对比模板", IsBuiltin: true, FieldsJSON: marshalFields(kdFields)},
			{ID: 70003, UserID: 0, Name: "计算机视觉对比模板 (Computer Vision)", Domain: "Computer Vision", Description: "面向骨干网络、训练策略、分辨率与精度-算力权衡的视觉文献对比模板", IsBuiltin: true, FieldsJSON: marshalFields(cvFields)},
			{ID: 70004, UserID: 0, Name: "大语言模型对比模板 (Large Language Models)", Domain: "Large Language Models", Description: "面向模型参数量、预训练/微调方法、上下文窗口与评测基准的对比模板", IsBuiltin: true, FieldsJSON: marshalFields(llmFields)},
		}
		for _, t := range templates {
			s.db.WithContext(ctx).Create(&t)
		}
	}

	const demoUserID int64 = 10001
	var compCount int64
	s.db.WithContext(ctx).Model(&model.Comparison{}).Where("user_id = ?", demoUserID).Count(&compCount)
	if compCount > 0 {
		return
	}

	var flTpl model.ComparisonTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", 70001).First(&flTpl).Error; err != nil {
		return
	}

	comp := &model.Comparison{
		ID:            80001,
		UserID:        demoUserID,
		Title:         "联邦学习 Non-IID 与异构知识传递代表作横向对比 (FedAvg / FedProx / MOON / FedProto / FedGen)",
		ResearchTopic: "Federated Learning & Prototype Knowledge Transfer",
		Description:   "围绕统计异质性（Non-IID）与系统/模型异质性，系统对照参数聚合、近端正则、模型对比学习、类别原型传递与无数据生成蒸馏五种核心范式。",
		TemplateID:    70001,
		TemplateName:  flTpl.Name,
		Status:        "completed",
		FieldsJSON:    flTpl.FieldsJSON,
	}
	s.db.WithContext(ctx).Create(comp)

	paperIDs := []int64{40001, 40002, 40003, 40004, 40005}
	for idx, pid := range paperIDs {
		s.db.WithContext(ctx).Create(&model.ComparisonPaper{
			ComparisonID: comp.ID,
			PaperID:      pid,
			DisplayOrder: int32(idx + 1),
		})
	}

	// Auto populate from paper_analysis + specialized FL fields
	s.populateComparisonFromAnalysis(ctx, comp.ID, paperIDs, unmarshalFields(flTpl.FieldsJSON), true)

	// Add specialized fields (model_heterogeneity, knowledge_transfer, comm_cost) with exact page evidence
	specialCells := []struct {
		PaperID int64
		Key     string
		Val     string
		Page    int32
		Sec     string
		Quote   string
	}{
		{40001, "model_heterogeneity", "仅支持同构客户端模型（Homogeneous CNN / LSTM）", 2, "Methodology", "The central server aggregates client weights via weighted averaging: w_{t+1} = sum (n_k/n) w_{t+1}^k."},
		{40001, "knowledge_transfer", "全模型参数加权平均 (Parameter Averaging)", 2, "Methodology", "learns a shared model by aggregating locally-computed updates via FederatedAveraging."},
		{40001, "comm_cost", "单轮需传输完整模型参数，但总通信轮数比 FedSGD 减少 10x–100x", 2, "Methodology", "performing multiple local SGD updates dramatically reduces required communication rounds by 10x to 100x."},

		{40002, "model_heterogeneity", "仅支持同构模型结构，但允许算力受限客户端执行可变本地迭代（Partial Work）", 2, "Methodology", "FedProx allows variable local work gamma_k^t across devices depending on their available compute."},
		{40002, "knowledge_transfer", "带 L2 近端约束的全局模型参数聚合", 2, "Methodology", "h_k(w; w^t) = F_k(w) + (mu / 2) ||w - w^t||^2."},
		{40002, "comm_cost", "单轮传输完整模型权重，与 FedAvg 单轮开销相同但慢节点无需重传", 4, "Conclusion", "FedProx still relies on full homogeneous model parameter transmission."},

		{40003, "model_heterogeneity", "同构编码器 + 投影头架构（ResNet-50 / CNN + Projection Head）", 2, "Methodology", "Each network comprises a base encoder, a projection head mapping features to representation z, and an output classification layer."},
		{40003, "knowledge_transfer", "全局参数聚合 + 模型级表征对比约束 L_con(z, z_glob, z_prev)", 2, "Methodology", "maximizing agreement between current local representation and global representation while minimizing agreement with previous local representation."},
		{40003, "comm_cost", "单轮传输完整模型参数；客户端本地需额外驻留 2 个参考模型副本", 3, "Results", "Clients must store both the global model and their previous local model in GPU memory."},

		{40004, "model_heterogeneity", "原生支持异构客户端骨干网络（不同层数 CNN / ResNet18 混合部署）", 3, "Experiments", "with heterogeneous client backbones (ResNet18 and varying CNN depths)."},
		{40004, "knowledge_transfer", "紧凑类别原型向量双向传递 (Class Prototypes C_i^{(j)})", 2, "Methodology", "FedProto communicates compact class prototypes—the mean embedding vectors of samples belonging to the same class."},
		{40004, "comm_cost", "极低：0.04–0.15 MB/轮（相比 ResNet 全参数传输降低 >98%）", 3, "Experiments", "Communicating prototypes requires only 0.04 MB to 0.15 MB per round compared to >40 MB for full ResNet parameter transmission."},

		{40005, "model_heterogeneity", "支持异构特征提取器（仅需共享顶层隐特征空间与分类器头）", 2, "Methodology", "Clients sample synthetic latent features z_hat ~ G_w(z, y) to regularize their predictor heads."},
		{40005, "knowledge_transfer", "服务端训练并下发条件隐特征生成器 G_w(z|y) 进行无数据蒸馏", 2, "Methodology", "the server learns a lightweight conditional generator G(z | y) using ensemble predictions from client classifiers."},
		{40005, "comm_cost", "传输分类头与轻量隐空间生成器 G_w，无需外部公共数据集", 3, "Results", "operates in the compact latent feature space rather than pixel space, it adds minimal overhead."},
	}

	for _, sc := range specialCells {
		ev := []*protosdkws.Evidence{{Page: sc.Page, Section: sc.Sec, Quote: sc.Quote}}
		s.upsertCell(ctx, comp.ID, sc.PaperID, sc.Key, sc.Val, "reported", "explicit", ev, "system")
	}

	diff := &model.ComparisonAnalysis{
		ID:                  85001,
		ComparisonID:        comp.ID,
		ObjectiveComparison: "五篇论文均致力于解决联邦学习在非独立同分布（Non-IID）数据下的性能退化问题，但演进重心从早期的【降低通信轮数】(FedAvg, 2017) 与【容忍慢节点及抑制参数发散】(FedProx, 2020)，逐步转向【校准高层特征表征】(MOON, 2021) 以及【突破客户端模型同构限制与无数据知识迁移】(FedProto, 2022 & FedGen, 2021)。",
		MethodDifferences:   "1. 参数聚合路线：FedAvg 直接加权平均参数；FedProx 在本地目标增加二阶近端项 (μ/2)||w-w^t||^2；MOON 增加投影头并在表征空间计算模型级 InfoNCE 对比损失。\n2. 知识/原型传递路线：FedProto 完全摒弃参数传输，仅聚合与对齐同类样本的特征均值向量（Class Prototypes）；FedGen 则在服务端训练条件隐特征生成器 G(z|y)，客户端采样合成特征通过 KL 散度蒸馏分类头。",
		DatasetDifferences:  "均包含 MNIST 与 CIFAR-10 作为核心视觉基准；FedProx/FedProto/FedGen 进一步引入 62 类手写字符集 FEMNIST/EMNIST；MOON 侧重复杂视觉特征，额外评测了 CIFAR-100 与 Tiny-ImageNet (Dirichlet β=0.5)；FedAvg/FedProx 还覆盖了 Shakespeare 文本序列数据集。",
		SettingDifferences:  "客户端规模从 K=10 (MOON 深度视觉实验)、K=20 (FedProto/FedGen) 到 K=100–200 (FedAvg/FedProx) 不等；在骨干网络上，FedAvg/FedProx/MOON 严格要求所有客户端架构同构，而 FedProto 明确评测了不同深度 CNN 与 ResNet18 混合异构部署。",
		ResultDifferences:   "1. 通信效率方面：FedProto 单轮仅需 0.04–0.15 MB，相比全模型传输降低 98% 以上通信量；\n2. 视觉特征精度方面：MOON 在同构 CIFAR-100 上达到 68.8%（优于 FedAvg 64.5%、FedProx 65.2%）；\n3. 极端 Non-IID 与异构鲁棒性方面：FedProx 在 90% 慢节点下平均提升 22%；FedGen 在 Dirichlet α=0.05 极端异质且无辅助数据条件下超越 FedAvg 6.2%。注意各论文客户端采样率与 Non-IID 划分参数不同，不宜脱离实验设置直接按单一数字排序。",
		CommonLimitations:   "1. 参数级与对比学习方法（FedAvg/FedProx/MOON）受限于同构模型假设，且 MOON 使客户端显存占用翻倍；\n2. 原型与生成蒸馏方法（FedProto/FedGen）虽大幅降低通信并支持异构骨干，但要求隐特征维度对齐或服务端额外训练生成器，在跨任务语义剧烈偏移或超大规模类别空间下仍有待提升。",
		EvolutionTrend:      "2017 (FedAvg: 确立本地多步 SGD + 中心参数平均基线) → 2020 (FedProx: 引入近端正则化解决系统与统计双异质性) → 2021 (MOON & FedGen: 从参数空间走向特征表征对比与服务端隐空间无数据蒸馏) → 2022 (FedProto: 以超轻量类别原型替代权重传输，实现真正的异构客户端联邦协同)。",
		SummaryMarkdown: `### 联邦学习多论文横向对比核心结论

1. **技术路线演进脉络**：从 **传递模型参数**（FedAvg / FedProx）$\rightarrow$ **约束特征表征**（MOON）$\rightarrow$ **传递抽象知识/原型**（FedGen / FedProto）。
2. **选型建议**：
   - 若边缘设备算力充裕且模型同构、追求高精度视觉表征，优先选择 **MOON**；
   - 若存在大量异构硬件慢节点（Stragglers），选择 **FedProx**；
   - 若客户端模型架构异构且带宽极度受限，优先采用 **FedProto**（通信量降低 $>98\%$）；
   - 若处于极端 Non-IID（如 $\alpha=0.05$）且无法获取外部公开辅助数据集，推荐采用 **FedGen**。`,
		UpdatedAt: time.Now(),
	}
	s.db.WithContext(ctx).Create(diff)
}
