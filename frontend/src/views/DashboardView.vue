<template>
  <div class="dashboard-view">
    <div class="page-header">
      <div>
        <h1 class="page-title">PaperCompare · 主页</h1>
        <p class="page-subtitle">快捷查看历史论文对比记录，一键进入多论文横向对比与页码级证据溯源工作台 · 欢迎回来，{{ userProfile?.nickname || 'Researcher' }}</p>
      </div>
      <div class="page-actions">
        <router-link to="/comparisons?create=1" class="btn-primary">
          + 新建多论文对比
        </router-link>
        <router-link to="/comparisons" class="btn-secondary">
          进入论文对比中心 →
        </router-link>
      </div>
    </div>

    <!-- Concise 3-Metric Stat Grid -->
    <div class="stat-grid" style="grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));">
      <div class="stat-card" @click="$router.push('/comparisons')" style="cursor:pointer;">
        <span class="stat-label">历史对比记录总数</span>
        <span class="stat-value">{{ recentComparisons.length || overview.total_comparisons || 0 }}</span>
        <span class="stat-sub">支持 2–10 篇论文跨维度横向矩阵对比</span>
      </div>
      <div class="stat-card" @click="$router.push('/comparisons')" style="cursor:pointer;">
        <span class="stat-label">已入库参比论文</span>
        <span class="stat-value">{{ overview.total_papers || 0 }}</span>
        <span class="stat-sub">已完成 PDF 章节切分与 10 维要素提取</span>
      </div>
      <div class="stat-card" @click="$router.push('/comparisons/80001')" style="cursor:pointer;">
        <span class="stat-label">典型对比示范项目</span>
        <span class="stat-value">FedAvg–FedGen</span>
        <span class="stat-sub">点击直达 5 篇联邦学习代表作 10 维对比工作台 →</span>
      </div>
    </div>

    <!-- Main Historical Comparison Records Section -->
    <div class="glass-card" style="margin-bottom: 24px;">
      <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:12px; margin-bottom:18px;">
        <div>
          <h2 style="font-size:1.25rem; font-weight:800;">历史论文对比记录 (Comparison History)</h2>
          <p style="font-size:0.84rem; color:var(--text-secondary);">点击任意历史记录即可快速恢复横向对比矩阵、核验原文页码证据或导出综述表格</p>
        </div>
        <div style="display:flex; gap:10px; align-items:center;">
          <input
            v-model="searchKeyword"
            class="input-field"
            style="width:240px;"
            placeholder="搜索对比标题、主题或论文..."
          />
          <router-link to="/comparisons?create=1" class="btn-primary btn-sm">+ 新建对比</router-link>
        </div>
      </div>

      <div v-if="loading" style="padding:36px; text-align:center; color:var(--text-muted);">
        正在加载历史对比记录...
      </div>
      <div v-else-if="!filteredComparisons.length" class="empty-state" style="padding:36px;">
        暂无匹配的对比项目，点击右上角「+ 新建多论文对比」开启横向对比分析
      </div>
      <div v-else style="display:flex; flex-direction:column; gap:14px;">
        <div
          v-for="comp in filteredComparisons"
          :key="comp.id"
          style="padding:18px; border:1px solid var(--border-color); border-radius:14px; cursor:pointer; transition:all 0.2s;"
          class="history-card"
          @click="$router.push(`/comparisons/${comp.id}`)"
        >
          <div style="display:flex; justify-content:space-between; align-items:flex-start; gap:14px; margin-bottom:8px;">
            <div>
              <div style="display:flex; align-items:center; gap:10px; flex-wrap:wrap; margin-bottom:4px;">
                <strong style="font-size:1.08rem; font-weight:800;">{{ comp.title || comp.name }}</strong>
                <span class="badge badge-info">{{ (comp.papers || []).length }} 篇参比论文 · {{ (comp.fields || comp.dimensions || []).length }} 个维度</span>
                <span class="badge" v-if="comp.template_name">{{ comp.template_name }}</span>
              </div>
              <p style="font-size:0.85rem; color:var(--text-secondary); margin-bottom:6px;">
                {{ comp.description || comp.research_topic || '多论文横向对比与证据溯源分析' }}
              </p>
            </div>
            <div style="display:flex; gap:8px; flex-shrink:0;" @click.stop>
              <button class="btn-primary btn-sm" @click="$router.push(`/comparisons/${comp.id}`)">
                打开对比矩阵 →
              </button>
              <button class="btn-secondary btn-sm" @click="exportMatrix(comp.id, 'markdown')">
                导出 MD
              </button>
              <button class="btn-secondary btn-sm" @click="exportMatrix(comp.id, 'csv')">
                导出 CSV
              </button>
              <button class="btn-danger btn-sm" @click="removeComparison(comp)">
                删除
              </button>
            </div>
          </div>

          <div style="display:flex; flex-wrap:wrap; align-items:center; justify-content:space-between; gap:8px; margin-top:10px; padding-top:10px; border-top:1px solid var(--border-color);">
            <div style="display:flex; flex-wrap:wrap; gap:6px;">
              <span v-for="p in (comp.papers || [])" :key="p.paper_id || p.id" class="tag-pill">
                📄 {{ p.title.split(':')[0].slice(0, 38) }} ({{ p.publication_year || 'N/A' }})
              </span>
            </div>
            <span style="font-size:0.76rem; color:var(--text-muted);" v-if="comp.updated_at">
              更新于 {{ formatTime(comp.updated_at) }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import api, { unwrap } from '../api';
import { toast } from '../utils/toast';
import { confirmModal } from '../utils/confirm';

const loading = ref(false);
const overview = ref({});
const recentComparisons = ref([]);
const userProfile = ref(null);
const searchKeyword = ref('');

const filteredComparisons = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase();
  if (!kw) return recentComparisons.value;
  return recentComparisons.value.filter(c => {
    const titleMatch = (c.title || c.name || '').toLowerCase().includes(kw);
    const descMatch = (c.description || c.research_topic || '').toLowerCase().includes(kw);
    const paperMatch = (c.papers || []).some(p => (p.title || '').toLowerCase().includes(kw));
    return titleMatch || descMatch || paperMatch;
  });
});

const formatTime = (ts) => {
  if (!ts) return '';
  const d = new Date(ts * 1000);
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
};

const exportMatrix = (id, format) => {
  const token = localStorage.getItem('access_token') || '';
  const url = `/api/comparisons/${id}/export?format=${format}&token=${encodeURIComponent(token)}`;
  window.open(url, '_blank');
};

const removeComparison = async (comp) => {
  const ok = await confirmModal('删除对比记录', `确定删除对比矩阵《${comp.title || comp.name}》吗？`);
  if (!ok) return;
  try {
    await api.delete(`/api/comparisons/${comp.id}`);
    recentComparisons.value = recentComparisons.value.filter(c => c.id !== comp.id);
    toast.success('已删除该条对比记录');
  } catch (e) {
    toast.error(e.message || '删除失败');
  }
};

const fetchDashboard = async () => {
  loading.value = true;
  try {
    const res = await api.get('/api/dashboard/overview');
    const data = unwrap(res) || {};
    overview.value = data.overview || {};
    recentComparisons.value = data.recent_comparisons || [];
    userProfile.value = data.user || null;
  } catch (e) {
    console.warn('Dashboard load failed:', e);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchDashboard();
  window.addEventListener('auth-updated', fetchDashboard);
});
</script>

<style scoped>
.history-card:hover {
  background: var(--surface-hover);
  transform: translateY(-1px);
}
</style>
