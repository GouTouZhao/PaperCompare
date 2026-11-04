<template>
  <div class="comparison-workspace-view" v-if="comp">
    <!-- Top Nav Header -->
    <div class="workspace-header">
      <div class="header-left">
        <button class="btn-back" @click="$router.push('/comparisons')">
          <span class="back-arrow">←</span> 返回对比列表
        </button>
        <div class="header-titles">
          <h1 class="comp-title">{{ comp.title || comp.name }}</h1>
          <p class="comp-subtitle" v-if="comp.description || comp.research_topic">
            {{ comp.description || comp.research_topic }}
          </p>
        </div>
      </div>

      <div class="header-actions">
        <button class="btn-ai-glow" :disabled="busy" @click="triggerAutoFill">
          <span class="icon">⚡</span>
          <span>{{ busy ? 'AI 运算中...' : 'AI 智能填充与核对' }}</span>
        </button>
        <button class="btn-outline-action" :disabled="busy" @click="triggerAnalyzeDiff">
          <span class="icon">🧠</span>
          <span>AI 深度差异分析</span>
        </button>
        <button class="btn-outline-action" @click="exportMatrix('markdown')">
          <span class="icon">⬇</span>
          <span>导出 Markdown</span>
        </button>
        <button class="btn-outline-action" @click="exportMatrix('csv')">
          <span class="icon">📊</span>
          <span>导出 Excel (CSV)</span>
        </button>
      </div>
    </div>

    <!-- Modular Stat Highlights Banner -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon">📚</div>
        <div class="stat-content">
          <div class="stat-value">{{ (comp.papers || []).length }} 篇</div>
          <div class="stat-label">横向参比论文</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">🎯</div>
        <div class="stat-content">
          <div class="stat-value">{{ (comp.fields || comp.dimensions || []).length }} 个</div>
          <div class="stat-label">对比科研维度</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">🛡️</div>
        <div class="stat-content">
          <div class="stat-value">{{ evidenceCoverageRate }}%</div>
          <div class="stat-label">原文证据链溯源率</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">🏷️</div>
        <div class="stat-content">
          <div class="stat-value">{{ comp.template_name || '标准对比模板' }}</div>
          <div class="stat-label">评测维度范式</div>
        </div>
      </div>
    </div>

    <!-- Filter & View Mode Controls Strip -->
    <div class="control-strip">
      <!-- Category Tabs -->
      <div class="category-tabs">
        <button
          v-for="cat in categoryTabs"
          :key="cat.id"
          class="cat-tab"
          :class="{ active: selectedCategory === cat.id }"
          @click="selectedCategory = cat.id"
        >
          {{ cat.name }} ({{ cat.count }})
        </button>
      </div>

      <!-- View Preferences -->
      <div class="view-preferences">
        <button
          class="pref-btn"
          :class="{ active: viewMode === 'compact' }"
          @click="viewMode = 'compact'"
          title="卡片精简模式：限制默认高度，便于横向扫读"
        >
          🧩 卡片速览
        </button>
        <button
          class="pref-btn"
          :class="{ active: viewMode === 'full' }"
          @click="viewMode = 'full'"
          title="全文详述模式：展开所有论述"
        >
          📜 完整展开
        </button>
        <label class="toggle-evidence">
          <input type="checkbox" v-model="showEvidence" />
          <span>显示原文引证</span>
        </label>
      </div>
    </div>

    <!-- Legend Bar -->
    <div class="legend-panel">
      <div class="legend-items">
        <span class="legend-label">证据置信度级别:</span>
        <span class="badge-tag reported">● reported 原文明确报告</span>
        <span class="badge-tag manual">● manual 人工核验修订</span>
        <span class="badge-tag inferred">● inferred 模型归纳推断</span>
        <span class="badge-tag not_reported">○ not_reported 原文未提及 (严禁幻觉)</span>
      </div>
      <div class="legend-hint">
        💡 点击单元格可随时人工修订；点击 <span class="highlight-link">📄 Page X</span> 可直接在内置阅读器中核验原文
      </div>
    </div>

    <!-- Horizontal Comparison Matrix Table -->
    <div class="matrix-card-container">
      <div class="matrix-scroll-wrapper">
        <table class="modern-matrix-table">
          <thead>
            <tr>
              <th class="th-dimension-header">
                <div class="th-dim-inner">
                  <span class="dim-icon">📊</span>
                  <span>对比维度 \ 参比论文</span>
                </div>
              </th>
              <th
                v-for="(paper, pIdx) in (comp.papers || [])"
                :key="paper.paper_id || paper.id || pIdx"
                class="th-paper-header"
              >
                <div class="paper-header-card">
                  <div class="paper-chip-index">PAPER {{ String(pIdx + 1).padStart(2, '0') }}</div>
                  <div
                    class="paper-header-title"
                    @click="openEvidencePreview(paper, 1, '')"
                    :title="'点击预览论文原文: ' + paper.title"
                  >
                    {{ paper.title }}
                  </div>
                  <div class="paper-header-meta">
                    <span v-if="paper.publication_year" class="meta-tag">{{ paper.publication_year }}</span>
                    <span class="meta-authors">{{ (paper.authors || []).slice(0, 2).join(', ') || '主要作者' }}</span>
                  </div>
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="dim in filteredDimensions"
              :key="dim.key"
              class="matrix-row"
            >
              <!-- Sticky Dimension Definition Column -->
              <td class="td-dimension">
                <div class="dim-info-card">
                  <div class="dim-title-row">
                    <span class="dim-icon-bullet">{{ getDimIcon(dim.key) }}</span>
                    <strong class="dim-name">{{ dim.label || dim.name }}</strong>
                  </div>
                  <p class="dim-desc" v-if="dim.description">{{ dim.description }}</p>
                </div>
              </td>

              <!-- Paper Content Cells -->
              <td
                v-for="paper in (comp.papers || [])"
                :key="(paper.paper_id || paper.id) + '_' + dim.key"
                class="td-cell"
                @click="openCellEditor(paper, dim)"
              >
                <div class="cell-card" :class="{ 'is-hoverable': true }">
                  <!-- Cell Header -->
                  <div class="cell-card-header">
                    <span
                      class="cell-badge"
                      :class="statusClass(cellData(paper, dim).value_status || cellData(paper, dim).status)"
                    >
                      {{ cellData(paper, dim).value_status || cellData(paper, dim).status || 'reported' }}
                    </span>
                    <button class="cell-edit-btn" title="修订单元格内容">
                      ✎ 修订
                    </button>
                  </div>

                  <!-- Cell Body Text -->
                  <div
                    class="cell-card-body"
                    :class="{ 'clamped': viewMode === 'compact' && !isCellExpanded(paper, dim) }"
                  >
                    {{ cellData(paper, dim).value || '未提及 (not_reported)' }}
                  </div>

                  <!-- Expand / Collapse Toggle for Long Content -->
                  <div
                    v-if="viewMode === 'compact' && isTextLong(cellData(paper, dim).value)"
                    class="cell-expand-action"
                    @click.stop="toggleCellExpand(paper, dim)"
                  >
                    <span>{{ isCellExpanded(paper, dim) ? '收起 ▴' : '展开全文 ▾' }}</span>
                  </div>

                  <!-- Evidence Footer Box -->
                  <div
                    v-if="showEvidence && cellData(paper, dim).evidence?.length"
                    class="cell-evidence-box"
                    @click.stop
                  >
                    <div
                      v-for="(ev, idx) in cellData(paper, dim).evidence"
                      :key="idx"
                      class="evidence-pill"
                      :title="ev.quote ? ('原文依据: ' + ev.quote) : '点击直接核验原文页码内容'"
                      @click="openEvidencePreview(paper, ev.page || 1, ev.quote || '')"
                    >
                      <span class="ev-page-icon">📄</span>
                      <span class="ev-page-text">P.{{ ev.page || 1 }}</span>
                      <span class="ev-section-text" v-if="ev.section">· {{ ev.section }}</span>
                    </div>
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Multi-Paper Synthesis & Difference Analysis Section -->
    <div class="synthesis-container" v-if="comp.diff_analysis">
      <div class="synthesis-header">
        <div class="synthesis-badge">SYNTHESIS & EVOLUTION</div>
        <h2 class="synthesis-title">🧠 多论文深度横向综合研判 (Cross-Paper Synthesis)</h2>
        <p class="synthesis-desc">
          涵盖核心方法路线、数据集分区分布、指标完备度、性能权衡与演进趋势的横向归纳
        </p>
      </div>

      <!-- 6 Modular Analysis Cards Grid -->
      <div class="synthesis-grid">
        <div class="diff-card card-accent-blue" v-if="comp.diff_analysis.objective_comparison">
          <div class="diff-card-head">
            <span class="diff-icon">🎯</span>
            <h3 class="diff-title">1. 研究目标与问题边界对比</h3>
          </div>
          <p class="diff-text">{{ comp.diff_analysis.objective_comparison }}</p>
        </div>

        <div class="diff-card card-accent-purple" v-if="comp.diff_analysis.method_differences">
          <div class="diff-card-head">
            <span class="diff-icon">🔬</span>
            <h3 class="diff-title">2. 核心架构与技术路线差异</h3>
          </div>
          <p class="diff-text">{{ comp.diff_analysis.method_differences }}</p>
        </div>

        <div class="diff-card card-accent-emerald" v-if="comp.diff_analysis.dataset_differences">
          <div class="diff-card-head">
            <span class="diff-icon">📊</span>
            <h3 class="diff-title">3. 实验数据集与异构划分策略</h3>
          </div>
          <p class="diff-text">{{ comp.diff_analysis.dataset_differences }}</p>
        </div>

        <div class="diff-card card-accent-amber" v-if="comp.diff_analysis.setting_differences || comp.diff_analysis.metric_differences">
          <div class="diff-card-head">
            <span class="diff-icon">⚙️</span>
            <h3 class="diff-title">4. 超参数设置与实验条件对比</h3>
          </div>
          <p class="diff-text">{{ comp.diff_analysis.setting_differences || comp.diff_analysis.metric_differences }}</p>
        </div>

        <div class="diff-card card-accent-cyan" v-if="comp.diff_analysis.result_differences">
          <div class="diff-card-head">
            <span class="diff-icon">📈</span>
            <h3 class="diff-title">5. 关键定量结果与性能权衡 (Trade-offs)</h3>
          </div>
          <p class="diff-text">{{ comp.diff_analysis.result_differences }}</p>
        </div>

        <div class="diff-card card-accent-rose" v-if="comp.diff_analysis.common_limitations || comp.diff_analysis.limitation_comparison">
          <div class="diff-card-head">
            <span class="diff-icon">⚠️</span>
            <h3 class="diff-title">6. 学术局限性与瓶颈横向对照</h3>
          </div>
          <p class="diff-text">{{ comp.diff_analysis.common_limitations || comp.diff_analysis.limitation_comparison }}</p>
        </div>

        <div class="diff-card card-accent-indigo card-span-2" v-if="comp.diff_analysis.evolution_trend">
          <div class="diff-card-head">
            <span class="diff-icon">🚀</span>
            <h3 class="diff-title">7. 领域技术脉络演进趋势</h3>
          </div>
          <p class="diff-text">{{ comp.diff_analysis.evolution_trend }}</p>
        </div>
      </div>

      <!-- Rendered Academic Markdown Related Work Section -->
      <div v-if="comp.diff_analysis.summary_markdown" class="related-work-panel">
        <div class="rw-header">
          <div class="rw-title-block">
            <span class="rw-icon">📝</span>
            <div>
              <h4 class="rw-title">综述级 Markdown 草稿 (可直接用于论文 Related Work 章节)</h4>
              <p class="rw-sub">基于以上结构化维度与差异研判自动排版，支持一键复制到 LaTeX / Markdown 论文中</p>
            </div>
          </div>
          <button class="btn-copy-md" @click="copyMarkdownReport">
            {{ copiedMd ? '✓ 已复制到剪贴板' : '📋 一键复制 Markdown' }}
          </button>
        </div>
        <div class="rw-body markdown-rendered" v-html="renderMarkdown(comp.diff_analysis.summary_markdown)"></div>
      </div>
    </div>

    <!-- Cell Editor Modal -->
    <div v-if="editingCell" class="modal-backdrop" @click.self="editingCell = null">
      <div class="modal-card-pro">
        <div class="modal-head">
          <h3 class="modal-title">修订对比矩阵单元格与原文依据</h3>
          <button class="modal-close-btn" @click="editingCell = null">✕</button>
        </div>
        <p class="modal-subtitle">
          论文：{{ editingCell.paperTitle }} · 维度：<strong>{{ editingCell.dimName }}</strong>
        </p>

        <div class="form-group-pro">
          <label class="form-label-pro">证据置信度状态</label>
          <select v-model="editingCell.status" class="select-pro">
            <option value="reported">reported (原文明确报告)</option>
            <option value="manual">manual (人工校对确认)</option>
            <option value="inferred">inferred (模型归纳推断)</option>
            <option value="not_reported">not_reported (原文未提及)</option>
          </select>
        </div>

        <div class="form-group-pro">
          <label class="form-label-pro">该维度结论论述 (建议 2-4 句清晰精炼文字)</label>
          <textarea v-model="editingCell.value" class="textarea-pro" rows="4"></textarea>
        </div>

        <div class="form-row-2">
          <div class="form-group-pro">
            <label class="form-label-pro">证据所在页码 (Page)</label>
            <input v-model.number="editingCell.page" type="number" min="1" class="input-pro" />
          </div>
          <div class="form-group-pro">
            <label class="form-label-pro">证据所在章节 (Section)</label>
            <input v-model="editingCell.section" class="input-pro" placeholder="如 3. Methodology 或 4. Experiments" />
          </div>
        </div>

        <div class="form-group-pro">
          <label class="form-label-pro">原文引用关键句 (Quote)</label>
          <input v-model="editingCell.quote" class="input-pro" placeholder="粘贴论文中的原始英文证据句子..." />
        </div>

        <div class="modal-foot">
          <button
            class="btn-open-reader"
            @click="openEvidencePreview({ paper_id: editingCell.paperId, title: editingCell.paperTitle }, editingCell.page || 1, editingCell.quote || '')"
          >
            核验原文 Page {{ editingCell.page || 1 }} →
          </button>
          <div class="foot-actions">
            <button class="btn-cancel" @click="editingCell = null">取消</button>
            <button class="btn-save" @click="saveCellEdit">保存更新</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Inline Evidence & PDF Page Preview Modal -->
    <div v-if="evidenceModal" class="modal-backdrop" @click.self="evidenceModal = null">
      <div class="modal-card-pro" style="max-width:760px;">
        <div class="modal-head">
          <h3 class="modal-title">📄 论文原文证据溯源核验</h3>
          <button class="modal-close-btn" @click="evidenceModal = null">✕</button>
        </div>
        <p class="modal-subtitle">
          <strong>{{ evidenceModal.paperTitle }}</strong>
        </p>

        <div v-if="evidenceModal.quote" style="padding:12px 14px; border-radius:10px; background:rgba(37,99,235,0.1); border-left:3px solid #2563eb; margin-bottom:14px; font-size:0.84rem;">
          <div style="font-weight:700; color:#2563eb; margin-bottom:4px;">当前引证关键句 (Page {{ evidenceModal.page }})：</div>
          <div style="font-style:italic;">"{{ evidenceModal.quote }}"</div>
        </div>

        <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:8px; margin-bottom:10px;">
          <div style="display:flex; gap:6px; flex-wrap:wrap;">
            <button
              v-for="pg in evidenceModal.pages"
              :key="pg.page_number"
              class="cat-tab"
              :class="{ active: evidenceModal.page === pg.page_number }"
              @click="evidenceModal.page = pg.page_number"
            >
              Page {{ pg.page_number }}
            </button>
          </div>
          <button class="btn-outline-action" @click="openRawPdf(evidenceModal.paperId, evidenceModal.page)">
            在新标签页打开原始 PDF (P.{{ evidenceModal.page }}) ↗
          </button>
        </div>

        <div style="max-height:360px; overflow-y:auto; padding:16px; border:1px solid var(--border-color); border-radius:12px; background:var(--surface-bg); font-size:0.86rem; line-height:1.75; white-space:pre-wrap;">
          {{ currentEvidencePageText }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import api, { unwrap } from '../api';
import { toast } from '../utils/toast';

const route = useRoute();
const router = useRouter();

const comp = ref(null);
const busy = ref(false);
const editingCell = ref(null);
const copiedMd = ref(false);

// View and filter states
const viewMode = ref('compact'); // 'compact' | 'full'
const showEvidence = ref(true);
const selectedCategory = ref('all');
const expandedCellKeys = ref(new Set());

const isTextLong = (txt) => {
  return txt && txt.length > 90;
};

const cellKey = (paper, dim) => {
  const pId = paper.paper_id || paper.id;
  return `${pId}_${dim.key}`;
};

const isCellExpanded = (paper, dim) => {
  return expandedCellKeys.value.has(cellKey(paper, dim));
};

const toggleCellExpand = (paper, dim) => {
  const k = cellKey(paper, dim);
  if (expandedCellKeys.value.has(k)) {
    expandedCellKeys.value.delete(k);
  } else {
    expandedCellKeys.value.add(k);
  }
};

const renderMarkdown = (md) => {
  if (!md) return '';
  return DOMPurify.sanitize(marked.parse(md));
};

const copyMarkdownReport = async () => {
  if (!comp.value?.diff_analysis?.summary_markdown) return;
  try {
    await navigator.clipboard.writeText(comp.value.diff_analysis.summary_markdown);
    copiedMd.value = true;
    toast.success('已复制综述级 Markdown 到剪贴板');
    setTimeout(() => { copiedMd.value = false; }, 2500);
  } catch (err) {
    toast.error('复制失败，请手动选取复制');
  }
};

const getDimIcon = (key) => {
  const map = {
    research_problem: '🎯',
    model_heterogeneity: '🧩',
    knowledge_transfer: '🔄',
    methodology: '🔬',
    datasets: '📊',
    exp_settings: '⚙️',
    metrics: '📏',
    results: '📈',
    comm_cost: '⚡',
    limitations: '⚠️',
  };
  return map[key] || '📌';
};

const categoryTabs = computed(() => {
  const allDims = comp.value?.fields || comp.value?.dimensions || [];
  return [
    { id: 'all', name: '全部维度', count: allDims.length },
    { id: 'method', name: '算法与架构', count: allDims.filter(d => ['research_problem', 'model_heterogeneity', 'knowledge_transfer', 'methodology'].includes(d.key)).length || 4 },
    { id: 'experiment', name: '数据与设置', count: allDims.filter(d => ['datasets', 'exp_settings'].includes(d.key)).length || 2 },
    { id: 'results', name: '指标与结果', count: allDims.filter(d => ['metrics', 'results'].includes(d.key)).length || 2 },
    { id: 'limitations', name: '开销与局限', count: allDims.filter(d => ['comm_cost', 'limitations'].includes(d.key)).length || 2 },
  ];
});

const filteredDimensions = computed(() => {
  const all = comp.value?.fields || comp.value?.dimensions || [];
  if (selectedCategory.value === 'all') return all;
  if (selectedCategory.value === 'method') {
    return all.filter(d => ['research_problem', 'model_heterogeneity', 'knowledge_transfer', 'methodology'].includes(d.key));
  }
  if (selectedCategory.value === 'experiment') {
    return all.filter(d => ['datasets', 'exp_settings'].includes(d.key));
  }
  if (selectedCategory.value === 'results') {
    return all.filter(d => ['metrics', 'results'].includes(d.key));
  }
  if (selectedCategory.value === 'limitations') {
    return all.filter(d => ['comm_cost', 'limitations'].includes(d.key));
  }
  return all;
});

const evidenceCoverageRate = computed(() => {
  if (!comp.value?.cells || comp.value.cells.length === 0) return 0;
  const withEv = comp.value.cells.filter(c => (c.evidence || []).length > 0 || c.value_status === 'reported');
  return Math.min(100, Math.round((withEv.length / comp.value.cells.length) * 100));
});

const loadComparison = async () => {
  try {
    const res = await api.get(`/api/comparisons/${route.params.id}`);
    comp.value = unwrap(res);
  } catch (e) {
    toast.error(e.message || '加载对比矩阵失败');
  }
};

const cellData = (paper, dim) => {
  if (!comp.value?.cells) {
    return { value: '未提及 (not_reported)', value_status: 'not_reported', status: 'not_reported', evidence: [] };
  }
  const pid = String(paper.paper_id || paper.id);
  const found = comp.value.cells.find(c =>
    String(c.paper_id) === pid && (c.field_key === dim.key || c.dimension_key === dim.key)
  );
  return found || {
    paper_id: pid,
    field_key: dim.key,
    value: '未提及 (not_reported)',
    value_status: 'not_reported',
    status: 'not_reported',
    evidence: [],
  };
};

const statusClass = (st) => {
  if (st === 'reported') return 'badge-reported';
  if (st === 'manual') return 'badge-manual';
  if (st === 'inferred') return 'badge-inferred';
  return 'badge-not-reported';
};

const openCellEditor = (paper, dim) => {
  const cell = cellData(paper, dim);
  const firstEv = cell?.evidence?.[0] || {};
  const st = cell?.value_status || cell?.status;
  const pId = String(paper.paper_id || paper.id);
  editingCell.value = {
    paperId: pId,
    paperTitle: paper.title,
    dimKey: dim.key,
    dimName: dim.label || dim.name,
    value: cell?.value || '',
    status: st === 'not_reported' ? 'manual' : (st || 'manual'),
    page: firstEv.page || 1,
    section: firstEv.section || 'Main Text',
    quote: firstEv.quote || '',
  };
};

const saveCellEdit = async () => {
  if (!editingCell.value) return;
  try {
    const c = editingCell.value;
    await api.put(`/api/comparisons/${comp.value.id}/cell`, {
      paper_id: c.paperId,
      field_key: c.dimKey,
      dimension_key: c.dimKey,
      value: c.value,
      value_status: c.status,
      status: c.status,
      evidence: [
        {
          page: c.page || 1,
          section: c.section || 'Manual Verification',
          quote: c.quote || c.value.slice(0, 120),
        },
      ],
    });
    editingCell.value = null;
    await loadComparison();
    toast.success('单元格与页码证据已更新');
  } catch (e) {
    toast.error(e.message);
  }
};

const triggerAutoFill = async () => {
  busy.value = true;
  try {
    const res = await api.post(`/api/comparisons/${comp.value.id}/auto-fill`, {
      overwrite_existing: false,
    });
    comp.value = unwrap(res);
    toast.success('AI 已自动填充空白对比维度并关联页码证据');
  } catch (e) {
    toast.error(e.message);
  } finally {
    busy.value = false;
  }
};

const triggerAnalyzeDiff = async () => {
  busy.value = true;
  try {
    const res = await api.post(`/api/comparisons/${comp.value.id}/analyze`);
    comp.value = unwrap(res);
    toast.success('AI 深度差异分析已更新');
  } catch (e) {
    toast.error(e.message);
  } finally {
    busy.value = false;
  }
};

const exportMatrix = (format) => {
  const token = localStorage.getItem('access_token') || '';
  const url = `/api/comparisons/${comp.value.id}/export?format=${format}&token=${encodeURIComponent(token)}`;
  window.open(url, '_blank');
};

const evidenceModal = ref(null);

const currentEvidencePageText = computed(() => {
  if (!evidenceModal.value) return '';
  const pg = (evidenceModal.value.pages || []).find(p => p.page_number === evidenceModal.value.page);
  return pg ? (pg.clean_text || pg.raw_text || '该页暂无文本') : '正在加载页码文本...';
});

const openEvidencePreview = async (paper, pageNum = 1, quote = '') => {
  const pid = paper.paper_id || paper.id;
  evidenceModal.value = {
    paperId: pid,
    paperTitle: paper.title,
    page: pageNum || 1,
    quote: quote || '',
    pages: [],
  };
  try {
    const res = await api.get(`/api/papers/${pid}/pages`);
    const data = unwrap(res);
    if (evidenceModal.value && evidenceModal.value.paperId === pid) {
      evidenceModal.value.pages = data?.pages || [];
    }
  } catch (e) {
    toast.error(e.message || '加载论文页码内容失败');
  }
};

const openRawPdf = (paperId, pageNum = 1) => {
  const token = localStorage.getItem('access_token') || '';
  const url = `/api/papers/${paperId}/file?token=${encodeURIComponent(token)}#page=${pageNum}`;
  window.open(url, '_blank');
};

onMounted(loadComparison);
</script>

<style scoped>
.comparison-workspace-view {
  max-width: 1400px;
  margin: 0 auto;
  padding-bottom: 60px;
}

/* Header */
.workspace-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.btn-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 6px 14px;
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  width: fit-content;
  transition: background-color 0.2s;
}

.btn-back:hover {
  background: var(--surface-hover);
}

.comp-title {
  font-size: 1.85rem;
  font-weight: 800;
  letter-spacing: -0.5px;
  line-height: 1.25;
}

.comp-subtitle {
  font-size: 0.9rem;
  color: var(--muted-color);
  margin-top: 4px;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.btn-ai-glow {
  background: linear-gradient(135deg, #2563eb, #7c3aed);
  color: #fff;
  border: none;
  border-radius: 10px;
  padding: 9px 18px;
  font-weight: 700;
  font-size: 0.88rem;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.35);
  transition: transform 0.15s, opacity 0.2s;
}

.btn-ai-glow:hover {
  transform: translateY(-1px);
  opacity: 0.92;
}

.btn-outline-action {
  background: var(--surface-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 8px 14px;
  font-size: 0.85rem;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: background-color 0.2s, border-color 0.2s;
}

.btn-outline-action:hover {
  background: var(--surface-hover);
  border-color: var(--text-color);
}

.btn-primary-action {
  background: var(--text-color);
  color: var(--bg-color);
  border: none;
  border-radius: 10px;
  padding: 8px 16px;
  font-size: 0.85rem;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}

/* Stats Highlight Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  padding: 16px 20px;
  transition: transform 0.2s, box-shadow 0.2s;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.05);
}

.stat-icon {
  font-size: 1.8rem;
  background: var(--surface-hover);
  border-radius: 12px;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-value {
  font-size: 1.15rem;
  font-weight: 800;
  line-height: 1.2;
}

.stat-label {
  font-size: 0.78rem;
  color: var(--muted-color);
  margin-top: 3px;
}

/* Control & Filter Strip */
.control-strip {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px;
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  padding: 10px 16px;
  margin-bottom: 18px;
}

.category-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.cat-tab {
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 0.82rem;
  font-weight: 600;
  background: transparent;
  color: var(--muted-color);
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.2s;
}

.cat-tab:hover {
  background: var(--surface-hover);
  color: var(--text-color);
}

.cat-tab.active {
  background: var(--text-color);
  color: var(--bg-color);
  font-weight: 700;
}

.view-preferences {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pref-btn {
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 0.78rem;
  font-weight: 600;
  background: var(--surface-hover);
  color: var(--muted-color);
  cursor: pointer;
  border: 1px solid var(--border-color);
}

.pref-btn.active {
  background: var(--text-color);
  color: var(--bg-color);
  border-color: var(--text-color);
}

.toggle-evidence {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  cursor: pointer;
  color: var(--text-color);
}

/* Legend */
.legend-panel {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding: 10px 16px;
  background: var(--surface-bg);
  border-radius: 10px;
  border: 1px solid var(--border-color);
  margin-bottom: 22px;
  font-size: 0.78rem;
}

.legend-items {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.legend-label {
  font-weight: 700;
  color: var(--text-color);
}

.badge-tag {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge-tag.reported {
  background: rgba(5, 150, 105, 0.14);
  color: #059669;
}

.badge-tag.manual {
  background: rgba(37, 99, 235, 0.14);
  color: #2563eb;
}

.badge-tag.inferred {
  background: rgba(217, 119, 6, 0.14);
  color: #d97706;
}

.badge-tag.not_reported {
  background: rgba(128, 128, 128, 0.14);
  color: var(--muted-color);
}

.legend-hint {
  color: var(--muted-color);
}

.highlight-link {
  color: #2563eb;
  font-weight: 700;
  text-decoration: underline;
}

/* Matrix Table Styling */
.matrix-card-container {
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
  border-radius: 18px;
  overflow: hidden;
  margin-bottom: 36px;
}

.matrix-scroll-wrapper {
  overflow-x: auto;
  width: 100%;
}

.modern-matrix-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  min-width: 880px;
}

.th-dimension-header {
  position: sticky;
  left: 0;
  z-index: 3;
  width: 240px;
  min-width: 220px;
  background: var(--bg-color);
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  border-right: 1px solid var(--border-color);
  text-align: left;
}

.th-dim-inner {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--muted-color);
}

.th-paper-header {
  min-width: 320px;
  padding: 16px 18px;
  background: var(--surface-bg);
  border-bottom: 1px solid var(--border-color);
  border-right: 1px solid var(--border-color);
  text-align: left;
  vertical-align: top;
}

.paper-header-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.paper-chip-index {
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.5px;
  color: #2563eb;
}

.paper-header-title {
  font-size: 0.96rem;
  font-weight: 800;
  color: var(--text-color);
  line-height: 1.35;
  cursor: pointer;
  transition: color 0.2s;
}

.paper-header-title:hover {
  color: #2563eb;
}

.paper-header-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.78rem;
  color: var(--muted-color);
}

.meta-tag {
  background: var(--surface-hover);
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 700;
}

.meta-authors {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Table Rows & Cells */
.matrix-row:not(:last-child) .td-dimension,
.matrix-row:not(:last-child) .td-cell {
  border-bottom: 1px solid var(--border-color);
}

.td-dimension {
  position: sticky;
  left: 0;
  z-index: 2;
  background: var(--bg-color);
  padding: 18px 20px;
  border-right: 1px solid var(--border-color);
  vertical-align: top;
}

.dim-info-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dim-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dim-icon-bullet {
  font-size: 1.1rem;
}

.dim-name {
  font-size: 0.92rem;
  font-weight: 800;
  color: var(--text-color);
}

.dim-desc {
  font-size: 0.76rem;
  color: var(--muted-color);
  line-height: 1.45;
  margin-top: 2px;
}

.td-cell {
  padding: 14px 16px;
  border-right: 1px solid var(--border-color);
  vertical-align: top;
  cursor: pointer;
  background: var(--bg-color);
  transition: background-color 0.2s;
}

.td-cell:hover {
  background: var(--surface-hover);
}

.cell-card {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 8px;
}

.cell-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cell-badge {
  font-size: 0.72rem;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 6px;
}

.badge-reported {
  background: rgba(5, 150, 105, 0.12);
  color: #059669;
}

.badge-manual {
  background: rgba(37, 99, 235, 0.12);
  color: #2563eb;
}

.badge-inferred {
  background: rgba(217, 119, 6, 0.14);
  color: #d97706;
}

.badge-not-reported {
  background: rgba(128, 128, 128, 0.14);
  color: var(--muted-color);
}

.cell-edit-btn {
  font-size: 0.72rem;
  color: var(--muted-color);
  opacity: 0.6;
  background: transparent;
  cursor: pointer;
  transition: opacity 0.2s, color 0.2s;
}

.cell-card:hover .cell-edit-btn {
  opacity: 1;
  color: #2563eb;
}

.cell-card-body {
  font-size: 0.86rem;
  line-height: 1.65;
  color: var(--text-color);
  white-space: pre-line;
}

.cell-card-body.clamped {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.cell-expand-action {
  font-size: 0.75rem;
  font-weight: 700;
  color: #2563eb;
  cursor: pointer;
  margin-top: -2px;
}

.cell-expand-action:hover {
  text-decoration: underline;
}

.cell-evidence-box {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px dashed var(--border-color);
}

.evidence-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  background: var(--surface-hover);
  border: 1px solid var(--border-color);
  font-size: 0.74rem;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s, background-color 0.15s;
}

.evidence-pill:hover {
  background: rgba(37, 99, 235, 0.12);
  border-color: #2563eb;
  color: #2563eb;
}

.ev-page-icon {
  font-size: 0.8rem;
}

.ev-page-text {
  font-weight: 700;
}

.ev-section-text {
  color: var(--muted-color);
  font-size: 0.7rem;
}

/* Synthesis & Difference Analysis */
.synthesis-container {
  margin-top: 48px;
  padding-top: 36px;
  border-top: 2px dashed var(--border-color);
}

.synthesis-header {
  margin-bottom: 24px;
}

.synthesis-badge {
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.8px;
  color: #7c3aed;
  margin-bottom: 4px;
}

.synthesis-title {
  font-size: 1.45rem;
  font-weight: 800;
  letter-spacing: -0.4px;
}

.synthesis-desc {
  font-size: 0.86rem;
  color: var(--muted-color);
  margin-top: 4px;
}

.synthesis-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin-bottom: 28px;
}

.card-span-2 {
  grid-column: span 2;
}

.diff-card {
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
  border-radius: 18px;
  padding: 22px;
  transition: transform 0.2s, box-shadow 0.2s;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.diff-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.06);
}

.card-accent-blue { border-top: 3px solid #2563eb; }
.card-accent-purple { border-top: 3px solid #9333ea; }
.card-accent-emerald { border-top: 3px solid #059669; }
.card-accent-amber { border-top: 3px solid #d97706; }
.card-accent-cyan { border-top: 3px solid #0891b2; }
.card-accent-rose { border-top: 3px solid #e11d48; }
.card-accent-indigo { border-top: 3px solid #4f46e5; }

.diff-card-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.diff-icon {
  font-size: 1.3rem;
}

.diff-title {
  font-size: 0.98rem;
  font-weight: 800;
  color: var(--text-color);
}

.diff-text {
  font-size: 0.88rem;
  line-height: 1.7;
  color: var(--muted-color);
}

/* Related Work Markdown Panel */
.related-work-panel {
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
  border-radius: 18px;
  padding: 24px;
}

.rw-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 18px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-color);
}

.rw-title-block {
  display: flex;
  align-items: center;
  gap: 12px;
}

.rw-icon {
  font-size: 1.5rem;
}

.rw-title {
  font-size: 1rem;
  font-weight: 800;
}

.rw-sub {
  font-size: 0.78rem;
  color: var(--muted-color);
}

.btn-copy-md {
  background: var(--text-color);
  color: var(--bg-color);
  border-radius: 8px;
  padding: 7px 14px;
  font-size: 0.82rem;
  font-weight: 700;
  cursor: pointer;
  transition: opacity 0.2s;
}

.btn-copy-md:hover {
  opacity: 0.88;
}

.markdown-rendered {
  font-size: 0.88rem;
  line-height: 1.75;
  color: var(--text-color);
}

.markdown-rendered :deep(h1),
.markdown-rendered :deep(h2),
.markdown-rendered :deep(h3) {
  margin-top: 14px;
  margin-bottom: 8px;
  font-weight: 800;
}

.markdown-rendered :deep(p) {
  margin-bottom: 10px;
}

.markdown-rendered :deep(ul),
.markdown-rendered :deep(ol) {
  margin-left: 20px;
  margin-bottom: 10px;
}

/* Pro Modal */
.modal-card-pro {
  background: var(--bg-color);
  color: var(--text-color);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  padding: 28px;
  width: 100%;
  max-width: 580px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.25);
}

.modal-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-title {
  font-size: 1.15rem;
  font-weight: 800;
}

.modal-close-btn {
  background: transparent;
  border: none;
  font-size: 1.1rem;
  color: var(--muted-color);
  cursor: pointer;
}

.modal-subtitle {
  font-size: 0.82rem;
  color: var(--muted-color);
  margin-top: 4px;
  margin-bottom: 18px;
}

.form-group-pro {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.form-label-pro {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--text-color);
}

.form-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.select-pro,
.input-pro,
.textarea-pro {
  width: 100%;
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 8px 12px;
  font-size: 0.88rem;
  background: var(--surface-bg);
  color: var(--text-color);
  outline: none;
}

.select-pro:focus,
.input-pro:focus,
.textarea-pro:focus {
  border-color: #2563eb;
}

.modal-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 14px;
  border-top: 1px solid var(--border-color);
}

.btn-open-reader {
  background: transparent;
  color: #2563eb;
  border: none;
  font-size: 0.82rem;
  font-weight: 700;
  cursor: pointer;
}

.btn-open-reader:hover {
  text-decoration: underline;
}

.foot-actions {
  display: flex;
  gap: 10px;
}

.btn-cancel {
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 7px 14px;
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
}

.btn-save {
  background: var(--text-color);
  color: var(--bg-color);
  border-radius: 8px;
  padding: 7px 16px;
  font-size: 0.84rem;
  font-weight: 700;
  cursor: pointer;
}

@media (max-width: 900px) {
  .synthesis-grid {
    grid-template-columns: 1fr;
  }
  .card-span-2 {
    grid-column: span 1;
  }
}
</style>
