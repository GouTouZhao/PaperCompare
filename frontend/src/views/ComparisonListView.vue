<template>
  <div class="comparison-list-view">
    <div class="page-header">
      <div>
        <h1 class="page-title">论文对比中心 (Paper Comparison)</h1>
        <p class="page-subtitle">上传参比 PDF 论文，支持 2–10 篇论文跨维度横向矩阵对比、页码级证据溯源、AI 差异归纳与综述级导出</p>
      </div>
      <div class="page-actions">
        <button class="btn-secondary" @click="triggerFileSelect" :disabled="uploading">
          {{ uploading ? '正在上传解析 PDF...' : '↑ 上传参比 PDF 论文' }}
        </button>
        <input ref="fileInput" type="file" accept=".pdf,application/pdf" style="display:none" @change="handleFileChange" />
        <button class="btn-primary" @click="openCreateModal">
          + 新建多论文对比项目
        </button>
      </div>
    </div>

    <!-- Built-in Domain Templates Showcase -->
    <div class="glass-card" style="margin-bottom:20px;">
      <div style="font-size:0.8rem; font-weight:800; text-transform:uppercase; color:var(--text-muted); margin-bottom:12px;">
        领域内置对比维度模板 (Comparison Templates)
      </div>
      <div class="grid-2">
        <div
          v-for="tpl in templates"
          :key="tpl.id"
          style="padding:14px; border:1px solid var(--border-color); border-radius:10px; background:var(--bg-primary); display:flex; flex-direction:column; justify-content:space-between;"
        >
          <div>
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:6px;">
              <strong style="font-size:0.95rem;">{{ tpl.name }}</strong>
              <span class="badge" :class="tpl.is_builtin || tpl.is_system ? 'badge-info' : 'badge-muted'">
                {{ tpl.is_builtin || tpl.is_system ? '系统内置模板' : '自定义模板' }} · {{ (tpl.fields || tpl.dimensions || []).length }} 维
              </span>
            </div>
            <p style="font-size:0.8rem; color:var(--text-secondary); margin-bottom:10px;">{{ tpl.description }}</p>
          </div>
          <div style="display:flex; flex-wrap:wrap; gap:6px;">
            <span v-for="d in (tpl.fields || tpl.dimensions || [])" :key="d.key" class="tag-pill">{{ d.label || d.name }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Existing Comparison Projects -->
    <div v-if="loading" class="glass-card" style="text-align:center; padding:40px;">正在加载对比矩阵项目...</div>
    <div v-else-if="!comparisons.length" class="glass-card empty-state">
      暂无对比项目，点击右上角「+ 新建多论文对比项目」即可开启横向对比分析
    </div>
    <div v-else style="display:flex; flex-direction:column; gap:16px; margin-bottom:24px;">
      <div
        v-for="comp in comparisons"
        :key="comp.id"
        class="glass-card"
        style="cursor:pointer;"
        @click="$router.push(`/comparisons/${comp.id}`)"
      >
        <div style="display:flex; justify-content:space-between; align-items:flex-start; gap:12px; margin-bottom:8px;">
          <div>
            <h3 style="font-size:1.15rem; font-weight:800; margin-bottom:4px;">{{ comp.title || comp.name }}</h3>
            <p style="font-size:0.86rem; color:var(--text-secondary);">{{ comp.description || comp.research_topic }}</p>
          </div>
          <div style="display:flex; gap:8px; flex-shrink:0;" @click.stop>
            <button class="btn-primary btn-sm" @click="$router.push(`/comparisons/${comp.id}`)">
              打开对比矩阵工作台 →
            </button>
            <button class="btn-danger btn-sm" @click="removeComparison(comp)">删除</button>
          </div>
        </div>

        <div style="display:flex; flex-wrap:wrap; gap:8px; margin-top:12px; padding-top:12px; border-top:1px solid var(--border-color);">
          <span class="badge badge-info">对比文献: {{ (comp.papers || []).length }} 篇</span>
          <span class="badge badge-success">对比维度: {{ (comp.fields || comp.dimensions || []).length }} 项</span>
          <span v-for="p in (comp.papers || [])" :key="p.paper_id || p.id" class="tag-pill">
            📄 {{ p.title.split(':')[0] }} ({{ p.publication_year }})
          </span>
        </div>
      </div>
    </div>

    <!-- Available Papers for Comparison -->
    <div class="glass-card">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:14px;">
        <div>
          <h3 style="font-size:1.05rem; font-weight:800;">候选参比论文池 ({{ libraryPapers.length }} 篇)</h3>
          <p style="font-size:0.8rem; color:var(--text-secondary);">支持直接上传 PDF 论文并自动完成章节切分与 10 维研究要素提取，供新建对比项目选用</p>
        </div>
        <button class="btn-secondary btn-sm" @click="triggerFileSelect" :disabled="uploading">
          + 上传新 PDF 论文
        </button>
      </div>
      <div style="display:flex; flex-direction:column; gap:8px;">
        <div
          v-for="p in libraryPapers"
          :key="p.id"
          style="display:flex; justify-content:space-between; align-items:center; gap:12px; padding:10px 14px; border:1px solid var(--border-color); border-radius:10px;"
        >
          <div style="min-width:0;">
            <div style="font-weight:700; font-size:0.9rem; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">
              {{ p.title }}
            </div>
            <div style="font-size:0.76rem; color:var(--text-secondary);">
              {{ (p.authors || []).join(', ') || '未知作者' }} · {{ p.publication_year || '-' }} · {{ p.journal || 'Academic Paper' }}
            </div>
          </div>
          <div style="display:flex; align-items:center; gap:8px; flex-shrink:0;">
            <span class="badge" :class="p.parse_status === 'completed' ? 'badge-success' : 'badge-warning'">
              {{ p.parse_status === 'completed' ? '已解析就绪' : p.parse_status }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Comparison Modal -->
    <div v-if="showCreateModal" class="modal-backdrop" @click.self="showCreateModal = false">
      <div class="modal-panel" style="max-width:680px;">
        <h3 style="font-size:1.2rem; font-weight:800; margin-bottom:6px;">新建多论文对比矩阵项目</h3>
        <p style="font-size:0.83rem; color:var(--text-secondary); margin-bottom:16px;">
          选择 2–10 篇论文与对比维度模板，系统将自动提取各篇论文的结构化结果与页码证据填入横向矩阵
        </p>

        <div class="form-group">
          <label class="form-label">对比项目名称</label>
          <input v-model="form.name" class="input-field" placeholder="例如：联邦学习经典算法演进与异构优化对比" />
        </div>

        <div class="form-group">
          <label class="form-label">对比目标与研究背景说明</label>
          <input v-model="form.description" class="input-field" placeholder="简述本次对比希望厘清的核心问题或选型目标..." />
        </div>

        <div class="form-group">
          <label class="form-label">选择对比维度模板</label>
          <select v-model="form.template_id" class="select-field">
            <option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
              {{ tpl.name }} ({{ (tpl.fields || tpl.dimensions || []).length }} 个维度)
            </option>
          </select>
        </div>

        <div class="form-group">
          <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:6px;">
            <label class="form-label" style="margin-bottom:0;">选择参与对比的论文 (已选 {{ form.paper_ids.length }} 篇，需选 2–10 篇)</label>
            <button type="button" class="btn-secondary btn-sm" @click="triggerFileSelect" :disabled="uploading">
              {{ uploading ? '上传中...' : '+ 上传本地 PDF' }}
            </button>
          </div>
          <div style="max-height:220px; overflow-y:auto; border:1px solid var(--border-color); border-radius:8px; padding:10px; display:flex; flex-direction:column; gap:8px;">
            <label
              v-for="p in libraryPapers"
              :key="p.id"
              style="display:flex; align-items:center; gap:10px; cursor:pointer; font-size:0.86rem;"
            >
              <input type="checkbox" :value="p.id" v-model="form.paper_ids" />
              <span style="font-weight:600;">{{ p.title }}</span>
              <span style="margin-left:auto; font-size:0.75rem; color:var(--text-muted);">{{ p.publication_year }}</span>
            </label>
          </div>
        </div>

        <div style="display:flex; align-items:center; justify-content:space-between; margin-top:16px;">
          <label style="display:flex; align-items:center; gap:8px; font-size:0.85rem; cursor:pointer;">
            <input type="checkbox" v-model="form.auto_fill" />
            <span>创建后立即执行 AI 结构化证据自动填充与差异分析</span>
          </label>
          <div style="display:flex; gap:8px;">
            <button class="btn-secondary" @click="showCreateModal = false">取消</button>
            <button class="btn-primary" :disabled="creating" @click="submitCreate">
              {{ creating ? '正在生成对比矩阵...' : '创建并进入工作台' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api, { unwrap } from '../api';
import { toast } from '../utils/toast';
import { confirmModal } from '../utils/confirm';

const route = useRoute();
const router = useRouter();

const comparisons = ref([]);
const templates = ref([]);
const libraryPapers = ref([]);
const loading = ref(false);
const showCreateModal = ref(false);
const creating = ref(false);
const uploading = ref(false);
const fileInput = ref(null);

const form = reactive({
  name: '',
  description: '',
  template_id: 0,
  paper_ids: [],
  auto_fill: true,
});

const triggerFileSelect = () => {
  if (fileInput.value) fileInput.value.click();
};

const handleFileChange = async (e) => {
  const file = e.target.files?.[0];
  if (!file) return;
  e.target.value = '';
  uploading.value = true;
  try {
    const fd = new FormData();
    fd.append('file', file);
    fd.append('auto_analyze', 'true');
    const res = await api.post('/api/papers/upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    const data = unwrap(res);
    if (data?.duplicate_found && data?.duplicate_paper) {
      toast.info(`该 PDF 已存在：${data.duplicate_paper.title}`);
    } else if (data?.paper) {
      toast.success(`已上传并解析论文：${data.paper.title}`);
      await loadData();
      if (!form.paper_ids.includes(data.paper.id)) {
        form.paper_ids.push(data.paper.id);
      }
    }
  } catch (err) {
    toast.error(err.message || '上传 PDF 失败');
  } finally {
    uploading.value = false;
  }
};

const loadData = async () => {
  loading.value = true;
  try {
    const [compRes, tplRes, paperRes] = await Promise.all([
      api.get('/api/comparisons'),
      api.get('/api/comparison-templates'),
      api.get('/api/papers', { params: { page_size: 100 } }),
    ]);
    comparisons.value = unwrap(compRes)?.list || [];
    templates.value = unwrap(tplRes)?.list || [];
    libraryPapers.value = unwrap(paperRes)?.list || [];

    if (templates.value.length && !form.template_id) {
      form.template_id = templates.value[0].id;
    }

    if (route.query.create === '1') {
      openCreateModal();
    } else if (route.query.paper_ids) {
      const ids = String(route.query.paper_ids).split(',').map(s => s.trim()).filter(Boolean);
      if (ids.length >= 2) {
        form.paper_ids = ids;
        form.name = `多论文专题对比 (${ids.length} 篇)`;
        showCreateModal.value = true;
      }
    }
  } catch (e) {
    toast.error(e.message || '加载对比项目失败');
  } finally {
    loading.value = false;
  }
};

const openCreateModal = () => {
  if (!form.paper_ids.length && libraryPapers.value.length >= 2) {
    form.paper_ids = libraryPapers.value.slice(0, 3).map(p => p.id);
  }
  showCreateModal.value = true;
};

const submitCreate = async () => {
  if (!form.name.trim()) return toast.warn('请输入对比项目名称');
  if (form.paper_ids.length < 2 || form.paper_ids.length > 10) {
    return toast.warn('请选择 2 到 10 篇参与对比的论文');
  }
  creating.value = true;
  try {
    const res = await api.post('/api/comparisons', {
      title: form.name,
      name: form.name,
      research_topic: form.description,
      description: form.description,
      template_id: form.template_id,
      paper_ids: form.paper_ids,
      auto_fill: form.auto_fill,
    });
    const created = unwrap(res);
    showCreateModal.value = false;
    toast.success('对比矩阵创建成功');
    router.push(`/comparisons/${created.id}`);
  } catch (e) {
    toast.error(e.message);
  } finally {
    creating.value = false;
  }
};

const removeComparison = async (comp) => {
  const ok = await confirmModal('删除对比项目', `确定删除对比矩阵《${comp.title || comp.name}》吗？`);
  if (!ok) return;
  try {
    await api.delete(`/api/comparisons/${comp.id}`);
    comparisons.value = comparisons.value.filter(c => c.id !== comp.id);
    toast.success('已删除对比项目');
  } catch (e) {
    toast.error(e.message);
  }
};

onMounted(loadData);
</script>
