# PaperCompare · 多论文横向对比与证据溯源平台 / Multi-Paper Comparison & Evidence Traceability Platform

<p align="center">
  <img src="frontend/public/logo.png" alt="PaperCompare Logo" width="120" height="120" />
</p>

<p align="center">
  <b>面向科研人员与课题组的轻量级多论文跨维度横向对比、页码级证据溯源与文献结构化工具</b><br>
  <i>A lightweight multi-paper cross-dimensional comparison and page-level evidence traceability platform for researchers.</i>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Language-Go%201.24%20%7C%20TypeScript-blue.svg" alt="Languages" />
  <img src="https://img.shields.io/badge/Frontend-Vue%203%20%2B%20Vite%20%2B%20Element%20Plus-green.svg" alt="Frontend" />
  <img src="https://img.shields.io/badge/Backend-gRPC%20%2B%20Protobuf%20%2B%20Gin-orange.svg" alt="Backend" />
  <img src="https://img.shields.io/badge/Orchestration-Kubernetes%20%2F%20Minikube%20%2F%20Docker-blueviolet.svg" alt="Orchestration" />
</p>

---

[中文文档](#中文文档) | [English Documentation](#english-documentation)

---

<a name="中文文档"></a>
## 🇨🇳 中文文档

### 一、项目概述

**PaperCompare** 是一款针对高校师生与科研人员在文献综述、方向调研场景下打造的轻量级论文横向对比与证据溯源平台。

在调研同一课题方向时，研究人员通常需要同时比对 3 至 10 篇代表性学术论文。传统的阅读流程需要反复切换多个 PDF 窗口，并在 Excel 或 Word 中手工搜集整理对比维度，既耗时费力，又容易因脱离上下文而丢失出处。

PaperCompare 聚焦于解决文献对比过程中的机械劳动：
1. **自动提取 10 维研读要素**：上传 PDF 论文后，系统自动解析并提取研究目标、核心方法、骨干架构、训练策略、基准数据集、评价指标、实验表现、创新点及局限性。
2. **多列并排对比矩阵**：将多篇论文在同一维度下的表现整齐对齐，直观展现不同算法演进与设计权衡。
3. **页码级原文证据溯源**：每一条提取结论后均附带明确的论文段落与精准页码引用（如 `Page 4, Section 3.2`），点击即可核实，避免 AI 幻觉造成的误导。
4. **历史记录与简报导出**：主页直接展示历史对比列表，对比矩阵支持一键导出为 Markdown 与 CSV 格式。

---

### 二、系统架构与技术选型

系统坚持**极简、轻量、高内聚**原则，去除冗余模块，核心由前端、API 网关及 3 个 gRPC 微服务构成：

```
                              ┌────────────────────────────────────────┐
                              │          Vue 3 前端 (Web UI)           │
                              │   - 主页 (历史记录)   - 论文对比工作台    │
                              └──────────────────┬─────────────────────┘
                                                 │ HTTP RESTful (Port 8080 -> 80)
                                                 ▼
                              ┌────────────────────────────────────────┐
                              │       paper-api (HTTP API Gateway)     │
                              │             Gin Web Framework          │
                              └───────┬──────────────┬──────────────┬──┘
                                      │ gRPC         │ gRPC         │ gRPC
                                      ▼              ▼              ▼
           ┌────────────────────────────┐ ┌──────────────────┐ ┌───────────────────────────┐
           │       paper-rpc-user       │ │ paper-rpc-paper  │ │     paper-rpc-compare     │
           │  (用户认证与 JWT 令牌管理)  │ │ (PDF解析/要素抽取│ │ (对比矩阵编排/证据溯源/导出)│
           └──────────────┬─────────────┘ └──────────┬───────┘ └─────────────┬─────────────┘
                          │                          │                       │
                          └──────────────────────────┼───────────────────────┘
                                                     ▼
                                      ┌──────────────────────────────┐
                                      │     MySQL 8.0 & Redis 缓存    │
                                      └──────────────────────────────┘
```

#### 技术栈说明
- **前端**：Vue 3 + Vite + Element Plus + Tailwind CSS + Pinia + Vue Router。
- **后端**：100% 纯 Go 语言实现（采用 gRPC + Protobuf 通信，Gin 提供统一 HTTP API 网关）。
- **数据持久化与缓存**：MySQL 8.0 持久化存储用户、论文元数据与对比矩阵；Redis 存储对比任务状态与缓存。
- **容器化部署**：支持 Minikube Kubernetes 集群编排与标准 Docker Compose。

---

### 三、微服务与端口定义

| 组件服务名 | 镜像名称 | 职责说明 | 内部端口 | 宿主机映射端口 |
| :--- | :--- | :--- | :--- | :--- |
| **paper-frontend** | `goutouz/paper-frontend:latest` | Vue 3 前端 + Nginx 反向代理 | `80` | `8080` |
| **paper-api** | `goutouz/paper-api:latest` | HTTP API 网关 (Gin) | `10002` | `10002` |
| **paper-rpc-user** | `goutouz/paper-rpc-user:latest` | 用户鉴权与 Token 管理 (gRPC) | `10003` | 集群内部访问 |
| **paper-rpc-paper** | `goutouz/paper-rpc-paper:latest` | PDF 上传/解析/要素提取 (gRPC) | `10004` | 集群内部访问 |
| **paper-rpc-compare** | `goutouz/paper-rpc-compare:latest` | 对比矩阵/证据溯源/导出 (gRPC) | `10006` | 集群内部访问 |
| **paper-redis** | `redis:alpine3.22` | 分布式缓存服务 | `6379` | `16379` |
| **MySQL** | 本地或容器化 MySQL 8.0 | 数据库持久化存储 | `3306` | `3306` |

---

### 四、快速上手与部署运行

#### 4.1 环境准备
- **操作系统**：Windows / Linux / macOS
- **开发工具**：Go 1.24+、Node.js 20+、Docker、Minikube 或 Docker Compose
- **依赖服务**：MySQL 8.0+、Redis 7+

#### 4.2 环境变量与安全配置
复制配置模板创建环境变量文件（请勿将真实私钥提交至代码仓库）：
```bash
cp .env.example .env
```
主要环境变量说明：
```env
# 大模型调用密钥 (DashScope / Qwen-Plus)
DASHSCOPE_API_KEY=<YOUR_DASHSCOPE_API_KEY>

# 数据库配置
MYSQL_HOST=host.minikube.internal
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=<YOUR_MYSQL_PASSWORD>
MYSQL_DB=papercompare
```

#### 4.3 方式一：Minikube 集群一键构建与部署（推荐）
1. 复制 Kubernetes 密钥模板并填入您的环境配置：
   ```bash
   cp backend/deployments/deploy/00-secret.yaml.example backend/deployments/deploy/00-secret.yaml
   ```
2. 运行一键构建脚本（在 Minikube 内部直接构建 Docker 镜像并部署）：
   ```cmd
   build_and_deploy_local.cmd
   ```
3. 暴露端口映射访问：
   ```cmd
   kubectl port-forward svc/paper-frontend-service 8080:80
   kubectl port-forward svc/paper-api-service 10002:10002
   ```
4. 打开浏览器访问：`http://localhost:8080`

#### 4.4 方式二：Docker Compose 快速启动
```bash
docker compose up -d --build
```
启动后访问 `http://localhost:8080` 即可。

#### 4.5 内置演示账号
- **账号**：`demo@papercompare.org`
- **密码**：`123456`
- **初始化数据**：系统启动时已内置联邦学习 5 篇经典论文对比案例（FedAvg, FedProx, SCAFFOLD, MOON, FedNova），可直接开箱体验并排矩阵与页码溯源。

---

### 五、目录结构

```text
PaperCompare/
├── backend/                        # 纯 Go 后端服务工程
│   ├── cmd/                        # 各微服务与网关启动入口
│   │   ├── api/                    # paper-api (HTTP API 网关)
│   │   └── rpc/                    # gRPC 微服务入口
│   │       ├── paper-rpc-user/     # 用户服务
│   │       ├── paper-rpc-paper/    # 论文解析服务
│   │       └── paper-rpc-compare/  # 对比分析服务
│   ├── config/                     # 基础服务配置文件
│   ├── deployments/deploy/         # Kubernetes 部署资源清单
│   │   ├── 00-infra.yaml           # Redis 部署与 RBAC 权限
│   │   ├── 00-secret.yaml.example  # 敏感配置模板（不含真实私钥）
│   │   ├── 01-rpc-services.yaml    # 3 个核心 RPC 微服务部署清单
│   │   └── 02-api-and-frontend.yaml# API 网关与前端 Nginx 部署清单
│   ├── internal/                   # 内部业务逻辑实现
│   └── pkg/                        # 公共库、存储层与数据库自动迁移
├── frontend/                       # Vue 3 前端工程
│   ├── src/
│   │   ├── views/                  # 核心页面：主页 (HomeView) 与论文对比 (CompareView)
│   │   ├── components/             # 对比矩阵、引用溯源抽屉、上传弹窗组件
│   │   └── api/                    # 前端 API 封装
│   └── package.json
├── docker-compose.yml              # Docker Compose 编排文件
├── build_and_deploy_local.cmd      # Minikube 本地一键构建与部署脚本
├── kubectl.cmd                     # Kubernetes 配置与部署重载脚本
├── HACKATHON_REPORT.md             # 黑客松赛事申报报告（精简务实版）
├── .gitignore                      # 敏感信息与运行产物过滤规则
└── README.md                       # 项目说明文档（中英双语）
```

---

### 六、隐私与安全声明

1. **敏感凭证防泄漏**：本项目严格遵循安全规范，禁止在代码库中硬编码私有 API 密钥与数据库凭据。所有敏感信息通过环境变量（如 `DASHSCOPE_API_KEY`、`MYSQL_PASSWORD`）或 Kubernetes Secret 注入。
2. **产物与数据隔离**：本地论文上传产生的临时 PDF 文件、存储产物、本地测试缓存已全部加入 `.gitignore` 规则，确保不被纳入版本控制。

---

<a name="english-documentation"></a>
## 🌐 English Documentation

### 1. Overview

**PaperCompare** is a lightweight academic tool designed for researchers and research labs to conduct **multi-paper cross-dimensional comparisons, page-level evidence traceability, and structured literature analysis**.

When conducting a literature review or exploring a new research direction, researchers typically examine 3 to 10 representative papers simultaneously. The traditional process involves manually toggling between multiple PDF windows, manually extracting key methodology dimensions, and maintaining comparison tables in Excel or Word. This manual process is tedious, error-prone, and often detaches claims from their original contexts.

PaperCompare streamlines this workflow:
1. **Automated 10-Dimensional Element Extraction**: Parses uploaded academic PDFs and extracts Research Objectives, Core Methods, Backbone Architectures, Key Assumptions, Training Strategies, Benchmark Datasets, Evaluation Metrics, Best Performance, Innovations, and Limitations.
2. **Side-by-Side Comparison Matrix**: Aligns multi-paper results side-by-side along identical dimensions for intuitive horizontal analysis.
3. **Page-Level Evidence Traceability**: Each extracted cell is accompanied by exact section and page-number citations (e.g., `Page 4, Section 3.2`), enabling one-click verification against the original PDF to eliminate LLM hallucinations.
4. **History Management & Export**: Built-in dashboard to quickly revisit saved comparisons and export matrices to Markdown and CSV formats.

---

### 2. Architecture & Tech Stack

PaperCompare follows a **minimalist, decoupled, and highly cohesive** architecture:

```
                              ┌────────────────────────────────────────┐
                              │         Vue 3 Frontend (Web UI)        │
                              │   - Dashboard (History)  - Compare Hub │
                              └──────────────────┬─────────────────────┘
                                                 │ HTTP RESTful (Port 8080 -> 80)
                                                 ▼
                              ┌────────────────────────────────────────┐
                              │       paper-api (HTTP API Gateway)     │
                              │             Gin Web Framework          │
                              └───────┬──────────────┬──────────────┬──┘
                                      │ gRPC         │ gRPC         │ gRPC
                                      ▼              ▼              ▼
           ┌────────────────────────────┐ ┌──────────────────┐ ┌───────────────────────────┐
           │       paper-rpc-user       │ │ paper-rpc-paper  │ │     paper-rpc-compare     │
           │ (User Auth & JWT Service)  │ │ (PDF Parse & Ext)│ │ (Matrix & Evidence Export)│
           └──────────────┬─────────────┘ └──────────┬───────┘ └─────────────┬─────────────┘
                          │                          │                       │
                          └──────────────────────────┼───────────────────────┘
                                                     ▼
                                      ┌──────────────────────────────┐
                                      │     MySQL 8.0 & Redis Cache  │
                                      └──────────────────────────────┘
```

#### Tech Stack
- **Frontend**: Vue 3, Vite, Element Plus, Tailwind CSS, Pinia, Vue Router.
- **Backend**: 100% Go implementation using gRPC & Protobuf for inter-service communication and Gin for the external HTTP API Gateway.
- **Persistence & Caching**: MySQL 8.0 for users, paper metadata, and comparison records; Redis for task status and caching.
- **Containerization**: Standard Docker Compose and Kubernetes (Minikube) manifests.

---

### 3. Services & Port Mapping

| Service | Image | Description | Internal Port | Host Port |
| :--- | :--- | :--- | :--- | :--- |
| **paper-frontend** | `goutouz/paper-frontend:latest` | Vue 3 UI + Nginx Reverse Proxy | `80` | `8080` |
| **paper-api** | `goutouz/paper-api:latest` | HTTP API Gateway (Gin) | `10002` | `10002` |
| **paper-rpc-user** | `goutouz/paper-rpc-user:latest` | User Auth & JWT RPC Service | `10003` | Cluster Internal |
| **paper-rpc-paper** | `goutouz/paper-rpc-paper:latest` | PDF Upload, Text & Dimension Extraction | `10004` | Cluster Internal |
| **paper-rpc-compare** | `goutouz/paper-rpc-compare:latest` | Comparison Matrix, Evidence & Export | `10006` | Cluster Internal |
| **paper-redis** | `redis:alpine3.22` | Distributed Cache | `6379` | `16379` |
| **MySQL** | Local or Containerized MySQL 8.0 | Primary Database | `3306` | `3306` |

---

### 4. Quick Start

#### 4.1 Prerequisites
- **OS**: Windows, macOS, or Linux
- **Tools**: Go 1.24+, Node.js 20+, Docker, Minikube or Docker Compose
- **Databases**: MySQL 8.0+, Redis 7+

#### 4.2 Configuration & Environment Variables
Copy the template file to set up your environment variables:
```bash
cp .env.example .env
```
Key configuration items:
```env
# LLM API Key (DashScope / Qwen-Plus)
DASHSCOPE_API_KEY=<YOUR_DASHSCOPE_API_KEY>

# Database Credentials
MYSQL_HOST=host.minikube.internal
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=<YOUR_MYSQL_PASSWORD>
MYSQL_DB=papercompare
```

#### 4.3 Method 1: Deploy with Minikube (Recommended)
1. Copy the Kubernetes secret template:
   ```bash
   cp backend/deployments/deploy/00-secret.yaml.example backend/deployments/deploy/00-secret.yaml
   ```
2. Run the one-click local build and deploy script:
   ```cmd
   build_and_deploy_local.cmd
   ```
3. Expose service port-forwarding:
   ```cmd
   kubectl port-forward svc/paper-frontend-service 8080:80
   kubectl port-forward svc/paper-api-service 10002:10002
   ```
4. Access the web interface at: `http://localhost:8080`

#### 4.4 Method 2: Deploy with Docker Compose
```bash
docker compose up -d --build
```
Open `http://localhost:8080` in your browser.

#### 4.5 Demo Account
- **Email**: `demo@papercompare.org`
- **Password**: `123456`
- **Preloaded Benchmark**: Includes 5 seminal Federated Learning papers (*FedAvg, FedProx, SCAFFOLD, MOON, FedNova*) with structured matrices and citation evidence ready for demonstration.

---

### 5. Repository Structure

```text
PaperCompare/
├── backend/                        # Go backend services
│   ├── cmd/                        # Main entrypoints for API and RPC services
│   ├── config/                     # Configuration definitions
│   ├── deployments/deploy/         # Kubernetes deployment manifests
│   ├── internal/                   # Core business logic
│   └── pkg/                        # Common libraries, database migration
├── frontend/                       # Vue 3 frontend
│   ├── src/
│   │   ├── views/                  # HomeView (History) and CompareView (Workbench)
│   │   ├── components/             # Matrix, Drawer, Upload Dialogs
│   │   └── api/                    # API client wrappers
│   └── package.json
├── docker-compose.yml              # Docker Compose service definition
├── build_and_deploy_local.cmd      # One-click Minikube image build & deployment
├── kubectl.cmd                     # Kubernetes manifest apply script
├── HACKATHON_REPORT.md             # Hackathon submission report (concise version)
├── .gitignore                      # Git ignore rules for artifacts & sensitive files
└── README.md                       # Project documentation (Bilingual)
```

---

### 6. Privacy & Security Notice

- **No Hardcoded Secrets**: All API keys, tokens, and database passwords are managed strictly via environment variables and Kubernetes Secrets. No credentials should ever be committed to git.
- **Artifact & Storage Isolation**: Uploaded paper PDFs, cache files, and local build outputs are ignored by `.gitignore` to protect privacy and repository cleanliness.
