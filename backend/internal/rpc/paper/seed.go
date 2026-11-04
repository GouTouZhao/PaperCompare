package paper

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	protopaper "github.com/GouTouZhao/PaperCompare/internal/protocol/paper"
	protosdkws "github.com/GouTouZhao/PaperCompare/internal/protocol/sdkws"
	"github.com/GouTouZhao/PaperCompare/pkg/common/storage/database/model"
	"github.com/GouTouZhao/PaperCompare/tools/pdfgen"
)

func (s *PaperServer) seedDemoPapers(ctx context.Context) {
	const demoUserID int64 = 10001
	var count int64
	s.db.WithContext(ctx).Model(&model.Paper{}).Where("user_id = ?", demoUserID).Count(&count)
	if count > 0 {
		return
	}

	type demoSpec struct {
		ID            int64
		Filename      string
		Title         string
		Authors       []string
		Year          int32
		Journal       string
		DOI           string
		URL           string
		Keywords      []string
		ReadingStatus string
		IsFavorite    bool
		Abstract      string
		Pages         []pdfgen.SamplePage
		Problem       *protopaper.AnalysisField
		Objective     *protopaper.AnalysisField
		Contributions *protopaper.AnalysisField
		Methodology   *protopaper.AnalysisField
		Datasets      *protopaper.AnalysisField
		ExpSettings   *protopaper.AnalysisField
		Metrics       *protopaper.AnalysisField
		Results       *protopaper.AnalysisField
		Limitations   *protopaper.AnalysisField
		FutureWork    *protopaper.AnalysisField
	}

	field := func(val, src, ver string, page int32, sec, quote string) *protopaper.AnalysisField {
		return &protopaper.AnalysisField{
			Value:              val,
			SourceType:         src,
			VerificationStatus: ver,
			Evidence: []*protosdkws.Evidence{
				{Page: page, Section: sec, Quote: quote},
			},
		}
	}

	specs := []demoSpec{
		{
			ID:            40001,
			Filename:      "FedAvg_McMahan_AISTATS2017.pdf",
			Title:         "Communication-Efficient Learning of Deep Networks from Decentralized Data (FedAvg)",
			Authors:       []string{"H. Brendan McMahan", "Eider Moore", "Daniel Ramage", "Seth Hampson", "Blaise Aguera y Arcas"},
			Year:          2017,
			Journal:       "AISTATS 2017",
			DOI:           "10.48550/arXiv.1602.05629",
			URL:           "https://arxiv.org/abs/1602.05629",
			Keywords:      []string{"Federated Learning", "FedAvg", "Decentralized Data", "Communication Efficiency"},
			ReadingStatus: "completed",
			IsFavorite:    true,
			Abstract:      "Modern mobile devices have access to a wealth of data suitable for learning models. We advocate an alternative that leaves the training data distributed on the mobile devices, and learns a shared model by aggregating locally-computed updates via FederatedAveraging (FedAvg).",
			Pages: []pdfgen.SamplePage{
				{
					Heading: "Abstract",
					Lines: []string{
						"Abstract: Modern mobile devices have access to a wealth of data suitable for learning models, which in turn can greatly improve the user experience on the device.",
						"We advocate an alternative that leaves the training data distributed on the mobile devices, and learns a shared model by aggregating locally-computed updates.",
						"1 Introduction: We term our approach Federated Learning. A principal advantage is decoupling model training from the need for direct access to raw training data.",
						"Federated optimization has several key properties that differentiate it from a typical distributed optimization problem: Non-IID data, unbalanced local sizes, massively distributed clients, and limited communication.",
					},
				},
				{
					Heading: "2 Methodology",
					Lines: []string{
						"The FederatedAveraging (FedAvg) algorithm combines local stochastic gradient descent (SGD) on each client with a server that performs model parameter averaging.",
						"In each communication round t, a fraction C of clients is selected. Each selected client k computes local gradients on its local dataset P_k for E local epochs with mini-batch size B.",
						"The central server aggregates client weights via weighted averaging: w_{t+1} = sum_{k=1}^K (n_k / n) w_{t+1}^k.",
						"Compared with FedSGD (E=1, B=infinity), performing multiple local SGD updates dramatically reduces required communication rounds by 10x to 100x.",
					},
				},
				{
					Heading: "3 Experiments",
					Lines: []string{
						"We evaluate FedAvg on MNIST digit recognition (CNN with two 5x5 convolution layers and 2NN), CIFAR-10 image classification, and a Shakespeare LSTM language model.",
						"Experimental Setup: K = 100 clients, client participation fraction C = 0.1 per round, local epochs E in {1, 5, 20}, local batch size B in {10, 50, infinity}, SGD optimizer.",
						"For Non-IID MNIST partitioning, we sort data by digit label and divide it into 200 shards of size 300, assigning each of 100 clients 2 shards so most clients only have 2 digits.",
					},
				},
				{
					Heading: "4 Results and Conclusion",
					Lines: []string{
						"Results: On MNIST CNN, FedAvg achieves 99.0% test accuracy in 10x to 95x fewer communication rounds than FedSGD. On CIFAR-10, FedAvg reaches 85.0% test accuracy within 2000 communication rounds.",
						"Limitations: Under pathological Non-IID partitions and heterogeneous client model architectures, weight averaging can experience client drift and accuracy degradation.",
						"Conclusion and Future Work: Future work will investigate differential privacy guarantees, secure multiparty aggregation, and stronger robustness under extreme Non-IID distributions.",
					},
				},
			},
			Problem:       field("移动终端隐私数据难以集中上传，且分布式边缘端存在显著的通信受限、数据非独立同分布（Non-IID）和样本量不均衡问题", "explicit", "verified", 1, "Introduction", "Federated optimization has several key properties: Non-IID data, unbalanced local sizes, massively distributed clients, and limited communication."),
			Objective:     field("在不收集原始隐私数据的前提下，通过多步本地 SGD 与中心参数加权平均显著降低分布式训练的通信轮数", "explicit", "verified", 1, "Abstract", "leaves the training data distributed on the mobile devices, and learns a shared model by aggregating locally-computed updates."),
			Contributions: field("1. 提出联邦学习（Federated Learning）范式；2. 设计 FederatedAveraging (FedAvg) 算法；3. 在视觉与语言任务上验证相比同步 SGD 降低 10–100 倍通信开销", "explicit", "verified", 2, "Methodology", "performing multiple local SGD updates dramatically reduces required communication rounds by 10x to 100x."),
			Methodology:   field("客户端在本地数据上执行 E 轮小批量（Batch Size B）SGD 更新，服务器端按客户端样本数加权平均全局模型参数 w_{t+1} = sum (n_k/n) w_{t+1}^k，要求所有客户端与服务器模型同构", "explicit", "verified", 2, "Methodology", "The central server aggregates client weights via weighted averaging: w_{t+1} = sum_{k=1}^K (n_k / n) w_{t+1}^k."),
			Datasets:      field("MNIST (Non-IID 200 shards 划分)、CIFAR-10 图像分类、Shakespeare 莎士比亚角色文本数据集", "explicit", "verified", 3, "Experiments", "We evaluate FedAvg on MNIST digit recognition, CIFAR-10 image classification, and a Shakespeare LSTM language model."),
			ExpSettings:   field("K=100 客户端，每轮采样率 C=0.1，本地轮数 E∈{1,5,20}，批大小 B∈{10,50,∞}，同构 CNN / LSTM 架构，SGD 优化器", "explicit", "verified", 3, "Experiments", "K = 100 clients, client participation fraction C = 0.1 per round, local epochs E in {1, 5, 20}, local batch size B in {10, 50, infinity}."),
			Metrics:       field("Test Accuracy (测试集准确率)、达标所需通信轮数 (Communication Rounds)、Speedup 倍数", "explicit", "verified", 4, "Results", "FedAvg achieves 99.0% test accuracy in 10x to 95x fewer communication rounds than FedSGD."),
			Results:       field("MNIST CNN 达到 99.0% 准确率（通信轮数减少 10–95 倍）；CIFAR-10 达到 85.0% 测试准确率", "explicit", "verified", 4, "Results", "On CIFAR-10, FedAvg reaches 85.0% test accuracy within 2000 communication rounds."),
			Limitations:   field("当数据高度 Non-IID 时多步本地更新易引发客户端漂移（Client Drift），且严格依赖客户端模型结构同构，单轮仍需传输完整模型参数", "explicit", "verified", 4, "Conclusion", "Under pathological Non-IID partitions and heterogeneous client model architectures, weight averaging can experience client drift."),
			FutureWork:    field("结合差分隐私（Differential Privacy）、安全多方聚合（Secure Aggregation）及应对极端 Non-IID 的优化策略", "explicit", "verified", 4, "Conclusion", "Future work will investigate differential privacy guarantees, secure multiparty aggregation, and stronger robustness."),
		},
		{
			ID:            40002,
			Filename:      "FedProx_Li_MLSys2020.pdf",
			Title:         "Federated Optimization in Heterogeneous Networks (FedProx)",
			Authors:       []string{"Tian Li", "Anit Kumar Sahu", "Manzil Zaheer", "Maziar Sanjabi", "Ameet Talwalkar", "Virginia Smith"},
			Year:          2020,
			Journal:       "MLSys 2020",
			DOI:           "10.48550/arXiv.1812.06127",
			URL:           "https://arxiv.org/abs/1812.06127",
			Keywords:      []string{"Federated Learning", "FedProx", "Systems Heterogeneity", "Statistical Heterogeneity"},
			ReadingStatus: "completed",
			IsFavorite:    true,
			Abstract:      "Federated Learning faces two core challenges: systems heterogeneity (stragglers and variable device hardware) and statistical heterogeneity (non-identically distributed data). We propose FedProx, which adds a proximal term to restrict local updates close to the global model.",
			Pages: []pdfgen.SamplePage{
				{
					Heading: "Abstract",
					Lines: []string{
						"Abstract: Federated Learning faces two core challenges: systems heterogeneity and statistical heterogeneity across decentralized devices.",
						"1 Introduction: In FedAvg, dropping straggler devices that fail to complete E local epochs introduces bias, while excessive local updates on Non-IID data cause divergence.",
						"We propose FedProx, a generalization and re-parametrization of FedAvg that tolerates partial work and stabilizes convergence via a proximal regularization term.",
					},
				},
				{
					Heading: "2 Methodology",
					Lines: []string{
						"Instead of solving the local empirical risk F_k(w) exactly, each client k minimizes a regularized objective: h_k(w; w^t) = F_k(w) + (mu / 2) ||w - w^t||^2.",
						"The proximal term (mu / 2) ||w - w^t||^2 penalizes large deviations of local weights w from the global model w^t, mitigating client drift under Non-IID distributions.",
						"Furthermore, FedProx allows variable local work gamma_k^t across devices depending on their available compute and battery budget instead of dropping stragglers.",
					},
				},
				{
					Heading: "3 Experiments",
					Lines: []string{
						"Datasets: Synthetic Non-IID datasets, MNIST, Federated EMNIST (FEMNIST, 62 classes, 200 clients), Shakespeare, and Sent140.",
						"Experimental Setup: System heterogeneity simulated with 0%, 50%, and 90% straggler devices performing fewer local iterations; proximal hyperparameter mu tuned in {0.001, 0.01, 0.1, 1.0}.",
						"Metrics: Training loss stability, Test Accuracy across heterogeneous distributions.",
					},
				},
				{
					Heading: "4 Results and Conclusion",
					Lines: []string{
						"Results: Across highly heterogeneous settings (90% stragglers), FedProx improves absolute test accuracy by 22% on average compared to FedAvg and significantly reduces oscillation.",
						"Limitations: Selecting the optimal hyperparameter mu requires tuning, and FedProx still relies on full homogeneous model parameter transmission.",
						"Conclusion: Adding a proximal term provides theoretical convergence guarantees under bounded dissimilarity and practical resilience to device stragglers.",
					},
				},
			},
			Problem:       field("联邦网络同时面临系统异质性（慢节点/掉线设备被丢弃）与统计异质性（Non-IID 下多轮本地更新导致模型发散）两大难题", "explicit", "verified", 1, "Introduction", "Federated Learning faces two core challenges: systems heterogeneity and statistical heterogeneity."),
			Objective:     field("允许算力受限客户端提交不完全本地更新（Partial Work），并通过近端正则项限制本地更新偏离全局模型", "explicit", "verified", 2, "Methodology", "tolerates partial work and stabilizes convergence via a proximal regularization term."),
			Contributions: field("1. 允许设备执行可变数量的本地迭代而不直接丢弃慢节点；2. 引入近端项 (μ/2)||w - w^t||^2 抑制客户端漂移；3. 给出有界非相似性假设下的理论收敛证明", "explicit", "verified", 2, "Methodology", "each client k minimizes a regularized objective: h_k(w; w^t) = F_k(w) + (mu / 2) ||w - w^t||^2."),
			Methodology:   field("在客户端本地目标函数中加入 L2 近端正则项 h_k(w; w^t) = F_k(w) + (μ/2)||w - w^t||^2，服务器端仍采用同构模型参数平均", "explicit", "verified", 2, "Methodology", "The proximal term (mu / 2) ||w - w^t||^2 penalizes large deviations of local weights w from the global model w^t."),
			Datasets:      field("Synthetic Non-IID 合成数据、MNIST、FEMNIST (62类手写字符)、Shakespeare、Sent140", "explicit", "verified", 3, "Experiments", "Datasets: Synthetic Non-IID datasets, MNIST, Federated EMNIST (FEMNIST, 62 classes, 200 clients), Shakespeare, and Sent140."),
			ExpSettings:   field("200 客户端，模拟 0% / 50% / 90% 慢节点比例，超参数 μ∈{0.001, 0.01, 0.1, 1.0}，同构模型结构", "explicit", "verified", 3, "Experiments", "System heterogeneity simulated with 0%, 50%, and 90% straggler devices; proximal hyperparameter mu tuned in {0.001, 0.01, 0.1, 1.0}."),
			Metrics:       field("Testing Accuracy (测试集准确率)、Training Loss 收敛方差", "explicit", "verified", 3, "Experiments", "Metrics: Training loss stability, Test Accuracy across heterogeneous distributions."),
			Results:       field("在 90% 慢节点与高异质数据下，相比 FedAvg 平均提升 22% 测试准确率，收敛曲线显著更平稳", "explicit", "verified", 4, "Results", "FedProx improves absolute test accuracy by 22% on average compared to FedAvg."),
			Limitations:   field("超参数 μ 对不同数据集较为敏感需额外调优；未解决跨客户端模型架构异构与高维参数通信开销问题", "explicit", "verified", 4, "Conclusion", "Selecting the optimal hyperparameter mu requires tuning, and FedProx still relies on full homogeneous model parameter transmission."),
			FutureWork:    field("探索自适应调整近端系数 μ 的策略及轻量化通信机制", "inferred", "verified", 4, "Conclusion", "Adding a proximal term provides theoretical convergence guarantees under bounded dissimilarity."),
		},
		{
			ID:            40003,
			Filename:      "MOON_Li_CVPR2021.pdf",
			Title:         "Model-Contrastive Federated Learning (MOON)",
			Authors:       []string{"Qinbin Li", "Bingsheng He", "Dawn Song"},
			Year:          2021,
			Journal:       "CVPR 2021",
			DOI:           "10.1109/CVPR46437.2021.01057",
			URL:           "https://arxiv.org/abs/2103.16257",
			Keywords:      []string{"Federated Learning", "Contrastive Learning", "MOON", "Non-IID", "Computer Vision"},
			ReadingStatus: "reading",
			IsFavorite:    false,
			Abstract:      "Federated learning with non-IID data suffers from drift in local representations. We propose Model-Contrastive Federated Learning (MOON), which utilizes the similarity between model representations to correct the local training of individual parties.",
			Pages: []pdfgen.SamplePage{
				{
					Heading: "Abstract",
					Lines: []string{
						"Abstract: Federated learning enables multiple parties to collaboratively train a machine learning model without communicating local data.",
						"1 Introduction: Because global model representations are generally more generalizable than skewed local model representations, we propose Model-Contrastive Learning (MOON).",
						"MOON conducts contrastive learning at the model level by maximizing agreement between current local representation and global representation while minimizing agreement with previous local representation.",
					},
				},
				{
					Heading: "2 Methodology",
					Lines: []string{
						"Architecture: Each network comprises a base encoder (e.g., ResNet-50 or CNN), a projection head mapping features to representation z, and an output classification layer.",
						"Loss Function: L = L_sup(w_i^t; (x,y)) + mu * L_con(w_i^t; w_i^{t-1}; w^t; x), where L_con = -log( exp(sim(z, z_glob)/tau) / (exp(sim(z, z_glob)/tau) + exp(sim(z, z_prev)/tau)) ).",
						"Here z is current local representation, z_glob is global model representation, and z_prev is local representation from round t-1.",
					},
				},
				{
					Heading: "3 Experiments and Results",
					Lines: []string{
						"Datasets: CIFAR-10, CIFAR-100, and Tiny-ImageNet partitioned using Dirichlet distribution Dir(beta=0.5).",
						"Experimental Setup: K=10 clients, 100 communication rounds, local epochs E=10, batch size B=64, temperature tau=0.5, contrastive weight mu in {1, 5, 10}.",
						"Results: On CIFAR-100, MOON reaches 68.8% accuracy (vs. FedAvg 64.5% and FedProx 65.2%). On Tiny-ImageNet, MOON improves accuracy by 2.6% over baselines.",
						"Limitations: Clients must store both the global model and their previous local model w_i^{t-1} in GPU memory, increasing local memory and forward pass cost by ~2x.",
					},
				},
			},
			Problem:       field("Non-IID 数据分布下本地模型学习到的特征表征容易向局部偏置退化，单纯在参数空间约束（如 FedProx）无法直接校准高层语义特征空间", "explicit", "verified", 1, "Introduction", "global model representations are generally more generalizable than skewed local model representations."),
			Objective:     field("在特征表征层面引入模型级对比学习（Model-level Contrastive Learning），拉近当前本地模型与全局模型表征，推远上一轮有偏本地表征", "explicit", "verified", 1, "Introduction", "maximizing agreement between current local representation and global representation while minimizing agreement with previous local representation."),
			Contributions: field("1. 首次提出模型级对比学习框架 MOON；2. 设计结合交叉熵与 InfoNCE 表征对比的双损失目标；3. 在 CIFAR-10/100 和 Tiny-ImageNet 上显著超越 FedAvg/FedProx", "explicit", "verified", 2, "Methodology", "Loss Function: L = L_sup + mu * L_con."),
			Methodology:   field("网络由基础编码器 + 投影头（Projection Head）+ 分类头组成；联合优化分类损失 L_sup 与模型对比损失 L_con(z, z_glob, z_prev)，温度系数 τ=0.5", "explicit", "verified", 2, "Methodology", "Each network comprises a base encoder, a projection head mapping features to representation z, and an output classification layer."),
			Datasets:      field("CIFAR-10、CIFAR-100、Tiny-ImageNet（采用 Dirichlet 分布 Dir(β=0.5) 模拟 Non-IID 划分）", "explicit", "verified", 3, "Experiments", "Datasets: CIFAR-10, CIFAR-100, and Tiny-ImageNet partitioned using Dirichlet distribution Dir(beta=0.5)."),
			ExpSettings:   field("K=10 客户端，100 通信轮，本地轮数 E=10，Batch Size B=64，温度 τ=0.5，对比权重 μ∈{1,5,10}，ResNet-50 / CNN 骨干", "explicit", "verified", 3, "Experiments", "K=10 clients, 100 communication rounds, local epochs E=10, batch size B=64, temperature tau=0.5."),
			Metrics:       field("Top-1 Test Accuracy (测试准确率)、收敛速度", "explicit", "verified", 3, "Results", "On CIFAR-100, MOON reaches 68.8% accuracy."),
			Results:       field("CIFAR-100 达到 68.8% 准确率（优于 FedAvg 64.5%、FedProx 65.2%）；Tiny-ImageNet 相比基线提升 2.6%", "explicit", "verified", 3, "Results", "On CIFAR-100, MOON reaches 68.8% accuracy (vs. FedAvg 64.5% and FedProx 65.2%)."),
			Limitations:   field("客户端训练时需同时驻留当前模型、全局模型和上一轮本地模型并做三次前向传播，本地显存与计算开销显著增加", "explicit", "verified", 3, "Results", "Clients must store both the global model and their previous local model in GPU memory, increasing local memory and forward pass cost."),
			FutureWork:    field("压缩历史模型存储开销并扩展到跨模态与轻量化边缘设备场景", "inferred", "unverified", 3, "Results", "increasing local memory and forward pass cost by ~2x."),
		},
		{
			ID:            40004,
			Filename:      "FedProto_Tan_AAAI2022.pdf",
			Title:         "FedProto: Federated Prototype Learning across Heterogeneous Clients",
			Authors:       []string{"Yue Tan", "Guodong Long", "Lu Liu", "Tianyi Zhou", "Qinghua Lu", "Jing Jiang", "Chengqi Zhang"},
			Year:          2022,
			Journal:       "AAAI 2022",
			DOI:           "10.1609/aaai.v36i8.20819",
			URL:           "https://arxiv.org/abs/2105.00243",
			Keywords:      []string{"Federated Learning", "Prototype Learning", "FedProto", "Model Heterogeneity", "Non-IID"},
			ReadingStatus: "reading",
			IsFavorite:    true,
			Abstract:      "Real-world federated learning often encounters both statistical heterogeneity and model heterogeneity across clients. We propose FedProto, where clients and the server communicate abstract class prototypes instead of gradients or model parameters.",
			Pages: []pdfgen.SamplePage{
				{
					Heading: "Abstract",
					Lines: []string{
						"Abstract: Real-world federated learning encounters both statistical heterogeneity and model architecture heterogeneity across edge clients.",
						"1 Introduction: Traditional parameter-averaging methods fail when clients deploy heterogeneous neural architectures. We propose Federated Prototype Learning (FedProto).",
						"Instead of exchanging high-dimensional model gradients, FedProto communicates compact class prototypes—the mean embedding vectors of samples belonging to the same class.",
					},
				},
				{
					Heading: "2 Methodology",
					Lines: []string{
						"Definition: For client i and class j, local prototype C_i^{(j)} = (1 / |D_{i,j}|) sum_{(x,y) in D_{i,j}} f_i(x), where f_i is client i's heterogeneous feature extractor.",
						"Server Aggregation: The server aggregates class prototypes across clients possessing class j: C_bar^{(j)} = (1 / |N_j|) sum_{i in N_j} (|D_{i,j}| / N_j) C_i^{(j)}.",
						"Local Objective: L_i = L_CE(F_i(x), y) + lambda * || C_i^{(j)} - C_bar^{(j)} ||_2, aligning local class prototypes with global prototypes while supporting heterogeneous backbones.",
					},
				},
				{
					Heading: "3 Experiments and Results",
					Lines: []string{
						"Datasets: MNIST, FEMNIST, and CIFAR-10 under Non-IID n-way k-shot settings with heterogeneous client backbones (ResNet18 and varying CNN depths).",
						"Communication Cost: Communicating prototypes requires only 0.04 MB to 0.15 MB per round compared to >40 MB for full ResNet parameter transmission (over 98% reduction).",
						"Results: FedProto achieves 97.4% accuracy on MNIST and outperforms FedAvg, FedProx, and FedPer under both statistical and model heterogeneity.",
						"Limitations: Requires all clients' feature extractors to project into a shared embedding dimension d, and classes unseen by a client lack local prototypes.",
					},
				},
			},
			Problem:       field("传统基于参数聚合的联邦方法无法支持异构客户端模型架构（Model Heterogeneity），且全参数传输通信开销过大", "explicit", "verified", 1, "Introduction", "Traditional parameter-averaging methods fail when clients deploy heterogeneous neural architectures."),
			Objective:     field("以紧凑的类别原型（Class Prototypes，即同类样本特征均值向量）替代模型参数进行客户端-服务器双向知识传递", "explicit", "verified", 1, "Introduction", "FedProto communicates compact class prototypes—the mean embedding vectors of samples belonging to the same class."),
			Contributions: field("1. 提出无需传输模型权重与梯度的联邦原型学习框架 FedProto；2. 天然支持异构客户端骨干网络；3. 通信量相比全模型传输降低 98% 以上", "explicit", "verified", 2, "Methodology", "aligning local class prototypes with global prototypes while supporting heterogeneous backbones."),
			Methodology:   field("客户端计算各类别特征均值作为本地原型 C_i^{(j)} 上传服务器；服务器按样本数加权聚合得到全局原型 C̄^{(j)}；客户端通过 L = L_CE + λ||C_i^{(j)} - C̄^{(j)}||_2 对齐原型", "explicit", "verified", 2, "Methodology", "Local Objective: L_i = L_CE(F_i(x), y) + lambda * || C_i^{(j)} - C_bar^{(j)} ||_2."),
			Datasets:      field("MNIST、FEMNIST、CIFAR-10（Non-IID n-way k-shot 少样本异构划分）", "explicit", "verified", 3, "Experiments", "Datasets: MNIST, FEMNIST, and CIFAR-10 under Non-IID n-way k-shot settings."),
			ExpSettings:   field("K=20 客户端，支持异构骨干网络（不同深度 CNN 与 ResNet18），正则权重 λ=1.0，SGD 优化器", "explicit", "verified", 3, "Experiments", "with heterogeneous client backbones (ResNet18 and varying CNN depths)."),
			Metrics:       field("Local Test Accuracy (个性化准确率)、Communication Cost (MB/round 通信开销)", "explicit", "verified", 3, "Experiments", "Communicating prototypes requires only 0.04 MB to 0.15 MB per round."),
			Results:       field("MNIST 达到 97.4% 个性化准确率，单轮通信量仅 0.04–0.15 MB（降低 >98% 通信开销），在模型异构场景显著优于 FedAvg/FedProx", "explicit", "verified", 3, "Results", "FedProto achieves 97.4% accuracy on MNIST and outperforms FedAvg, FedProx, and FedPer."),
			Limitations:   field("要求不同异构骨干网络最终投影到相同维度的特征空间，且跨任务或语义空间剧烈偏移时原型对齐能力受限", "explicit", "verified", 3, "Results", "Requires all clients' feature extractors to project into a shared embedding dimension d."),
			FutureWork:    field("研究跨维度原型投影与无标签半监督联邦原型对齐", "inferred", "verified", 3, "Results", "classes unseen by a client lack local prototypes."),
		},
		{
			ID:            40005,
			Filename:      "FedGen_Zhu_ICML2021.pdf",
			Title:         "Data-Free Knowledge Distillation for Heterogeneous Federated Learning (FedGen)",
			Authors:       []string{"Zhuangdi Zhu", "Junyuan Hong", "Jiayu Zhou"},
			Year:          2021,
			Journal:       "ICML 2021",
			DOI:           "10.48550/arXiv.2105.10056",
			URL:           "https://arxiv.org/abs/2105.10056",
			Keywords:      []string{"Federated Learning", "Knowledge Distillation", "Data-Free Distillation", "FedGen"},
			ReadingStatus: "unread",
			IsFavorite:    false,
			Abstract:      "Existing federated distillation methods typically require an external public proxy dataset, which is often unavailable in privacy-sensitive domains. We propose FedGen, a data-free federated distillation framework that learns a lightweight feature generator on the server.",
			Pages: []pdfgen.SamplePage{
				{
					Heading: "Abstract",
					Lines: []string{
						"Abstract: Knowledge distillation has emerged as an effective remedy for Non-IID federated learning, but typically requires a public proxy dataset.",
						"1 Introduction: We propose FedGen, a Data-Free Knowledge Distillation approach for heterogeneous federated learning.",
						"In FedGen, the server learns a lightweight conditional generator G(z | y) using ensemble predictions from client classifiers without accessing any external data.",
					},
				},
				{
					Heading: "2 Methodology",
					Lines: []string{
						"Server Generator Training: The server trains a lightweight generator G_w(z, y) to synthesize latent feature representations that maximize ensemble logit agreement across clients.",
						"Broadcast & Local Distillation: The generator G_w is broadcast to clients. Clients sample synthetic latent features z_hat ~ G_w(z, y) to regularize their predictor heads via KL-divergence distillation loss.",
						"Because G_w operates in the compact latent feature space rather than pixel space, it adds minimal overhead and avoids leaking raw client images.",
					},
				},
				{
					Heading: "3 Experiments and Results",
					Lines: []string{
						"Datasets: MNIST, EMNIST, CELEBA, and CIFAR-10 under Dirichlet Dir(alpha in {0.05, 0.1, 1.0}) Non-IID settings.",
						"Experimental Setup: K=20 clients, active ratio 0.5, 200 communication rounds, MLP/CNN backbones, generator learning rate 0.001.",
						"Results: Under extreme Non-IID (alpha=0.05), FedGen outperforms FedAvg by 6.2% and FedProx by 5.4% on CIFAR-10 without requiring any proxy dataset.",
						"Limitations: Training the server-side conditional generator introduces additional server optimization steps and transmitting the generator module adds slight communication overhead.",
					},
				},
			},
			Problem:       field("传统联邦知识蒸馏方法严重依赖与私有数据分布相近的外部公共辅助数据集（Proxy Dataset），在医疗金融等隐私敏感场景难以获取", "explicit", "verified", 1, "Abstract", "Knowledge distillation typically requires a public proxy dataset, which is often unavailable in privacy-sensitive domains."),
			Objective:     field("构建无数据（Data-Free）联邦知识蒸馏框架，在服务端利用客户端预测集成训练轻量级条件特征生成器并下发辅助本地蒸馏", "explicit", "verified", 1, "Introduction", "the server learns a lightweight conditional generator G(z | y) using ensemble predictions from client classifiers."),
			Contributions: field("1. 提出无需外部公共数据的无数据联邦蒸馏算法 FedGen；2. 在隐特征空间而非像素空间训练轻量生成器；3. 在极端 Non-IID (α=0.05) 下大幅提升泛化能力", "explicit", "verified", 2, "Methodology", "Because G_w operates in the compact latent feature space rather than pixel space, it adds minimal overhead."),
			Methodology:   field("服务端利用客户端分类头集成分数训练条件特征生成器 G_w(z,y)；将轻量生成器广播给客户端，客户端采样合成特征并通过 KL 散度蒸馏正则化本地分类器", "explicit", "verified", 2, "Methodology", "Clients sample synthetic latent features z_hat ~ G_w(z, y) to regularize their predictor heads via KL-divergence distillation loss."),
			Datasets:      field("MNIST、EMNIST、CelebA、CIFAR-10（Dirichlet α∈{0.05, 0.1, 1.0} 极端异质划分）", "explicit", "verified", 3, "Experiments", "Datasets: MNIST, EMNIST, CELEBA, and CIFAR-10 under Dirichlet Dir(alpha in {0.05, 0.1, 1.0}) Non-IID settings."),
			ExpSettings:   field("K=20 客户端，每轮激活率 0.5，200 通信轮，生成模型学习率 0.001，CNN/MLP 骨干网络", "explicit", "verified", 3, "Experiments", "K=20 clients, active ratio 0.5, 200 communication rounds, MLP/CNN backbones, generator learning rate 0.001."),
			Metrics:       field("Top-1 Test Accuracy (测试准确率)、不同 Dirichlet α 下的鲁棒性、收敛通信轮数", "explicit", "verified", 3, "Results", "Under extreme Non-IID (alpha=0.05), FedGen outperforms FedAvg by 6.2%."),
			Results:       field("在极端 Non-IID (α=0.05) 的 CIFAR-10 上，无需任何辅助数据即超越 FedAvg 6.2%、超越 FedProx 5.4%", "explicit", "verified", 3, "Results", "FedGen outperforms FedAvg by 6.2% and FedProx by 5.4% on CIFAR-10 without requiring any proxy dataset."),
			Limitations:   field("服务端每轮需额外训练条件生成器 G_w，且下发生成器参数带来小幅额外通信与客户端采样计算开销", "explicit", "verified", 3, "Results", "Training the server-side conditional generator introduces additional server optimization steps."),
			FutureWork:    field("扩展至大规模类别空间（如 ImageNet-1K）下的高效隐空间生成蒸馏", "inferred", "verified", 3, "Results", "operates in the compact latent feature space."),
		},
	}

	for _, sp := range specs {
		pdfPath := filepath.Join(s.dataDir, "papers", "user_10001", sp.Filename)
		_ = pdfgen.GenerateSamplePDF(pdfPath, sp.Title, strings.Join(sp.Authors, ", "), int(sp.Year), sp.DOI, sp.Pages)
		hash, sz := computeFileSHA256(pdfPath)

		p := &model.Paper{
			ID:              sp.ID,
			UserID:          demoUserID,
			Title:           sp.Title,
			Abstract:        sp.Abstract,
			Authors:         marshalStringSlice(sp.Authors),
			PublicationYear: sp.Year,
			Journal:         sp.Journal,
			DOI:             sp.DOI,
			Language:        "English",
			PaperType:       "Research Paper",
			Keywords:        marshalStringSlice(sp.Keywords),
			PublicationURL:  sp.URL,
			ReadingStatus:   sp.ReadingStatus,
			ParseStatus:     "completed",
			IsFavorite:      sp.IsFavorite,
		}
		s.db.WithContext(ctx).Create(p)

		pf := &model.PaperFile{
			ID:               sp.ID + 100,
			PaperID:          sp.ID,
			UserID:           demoUserID,
			FilePath:         pdfPath,
			OriginalFilename: sp.Filename,
			FileSize:         sz,
			MimeType:         "application/pdf",
			SHA256:           hash,
			PageCount:        int32(len(sp.Pages)),
		}
		s.db.WithContext(ctx).Create(pf)

		for idx, pg := range sp.Pages {
			txt := pg.Heading + "\n" + strings.Join(pg.Lines, "\n")
			s.db.WithContext(ctx).Create(&model.PaperPage{
				ID:         sp.ID*10 + int64(idx+1),
				PaperID:    sp.ID,
				PageNumber: int32(idx + 1),
				RawText:    txt,
				CleanText:  txt,
			})
			s.db.WithContext(ctx).Create(&model.PaperSection{
				ID:           sp.ID*10 + int64(idx+1),
				PaperID:      sp.ID,
				SectionTitle: pg.Heading,
				SectionType:  pg.Heading,
				StartPage:    int32(idx + 1),
				EndPage:      int32(idx + 1),
				Content:      strings.Join(pg.Lines, "\n"),
				SectionOrder: int32(idx + 1),
			})
		}

		pa := &model.PaperAnalysis{
			ID:                sp.ID + 200,
			PaperID:           sp.ID,
			ResearchProblem:   marshalAnalysisField(sp.Problem),
			ResearchObjective: marshalAnalysisField(sp.Objective),
			Contributions:     marshalAnalysisField(sp.Contributions),
			Methodology:       marshalAnalysisField(sp.Methodology),
			Datasets:          marshalAnalysisField(sp.Datasets),
			ExpSettings:       marshalAnalysisField(sp.ExpSettings),
			Metrics:           marshalAnalysisField(sp.Metrics),
			Results:           marshalAnalysisField(sp.Results),
			Limitations:       marshalAnalysisField(sp.Limitations),
			FutureWork:        marshalAnalysisField(sp.FutureWork),
			AnalysisVersion:   1,
			UpdatedAt:         time.Now(),
		}
		s.db.WithContext(ctx).Create(pa)
	}
}
