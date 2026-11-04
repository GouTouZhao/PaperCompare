<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import api, { unwrap } from './api';
import ToastManager from './components/ToastManager.vue';
import ConfirmModal from './components/ConfirmModal.vue';
import LoginModal from './components/LoginModal.vue';
import { showToast } from './utils/toast';
import logoImg from './assets/logo.png';

const router = useRouter();
const route = useRoute();

const isDark = ref(false);
const isSidebarExpanded = ref(true);
const showLoginModal = ref(false);

const currentUser = ref({
  isLoggedIn: false,
  nickname: 'Dr. GouTou',
  email: 'demo@papercompare.org'
});

const navItems = [
  { path: '/dashboard', name: '主页', sub: 'Comparison History' },
  { path: '/comparisons', name: '论文对比', sub: 'Paper Comparison' },
];

const ensureDefaultDemoSession = async () => {
  const token = localStorage.getItem('access_token');
  if (token) {
    currentUser.value.isLoggedIn = true;
    currentUser.value.nickname = localStorage.getItem('user_nickname') || 'Dr. GouTou';
    currentUser.value.email = localStorage.getItem('user_email') || '';
    return;
  }
  try {
    const res = await api.post('/api/auth/login', {
      email: 'demo@papercompare.org',
      password: '123456'
    });
    const data = unwrap(res);
    if (data && data.access_token) {
      localStorage.setItem('access_token', data.access_token);
      localStorage.setItem('refresh_token', data.refresh_token || '');
      localStorage.setItem('user_id', String(data.user?.id || 10001));
      localStorage.setItem('user_email', data.user?.email || 'demo@papercompare.org');
      localStorage.setItem('user_nickname', data.user?.nickname || 'Dr. GouTou');
      currentUser.value.isLoggedIn = true;
      currentUser.value.nickname = data.user?.nickname || 'Dr. GouTou';
      currentUser.value.email = data.user?.email || 'demo@papercompare.org';
      window.dispatchEvent(new Event('auth-updated'));
    }
  } catch (e) {
    console.warn('Auto login skipped:', e);
  }
};

const checkAuth = () => {
  const token = localStorage.getItem('access_token');
  if (token) {
    currentUser.value.isLoggedIn = true;
    currentUser.value.nickname = localStorage.getItem('user_nickname') || '研究员';
    currentUser.value.email = localStorage.getItem('user_email') || '';
  } else {
    currentUser.value.isLoggedIn = false;
    currentUser.value.nickname = '未登录';
    currentUser.value.email = '';
  }
};

const handleAuthExpired = () => {
  localStorage.removeItem('access_token');
  checkAuth();
  showLoginModal.value = true;
};

onMounted(async () => {
  await ensureDefaultDemoSession();
  checkAuth();
  window.addEventListener('auth-expired', handleAuthExpired);
  window.addEventListener('auth-updated', checkAuth);
});

onUnmounted(() => {
  window.removeEventListener('auth-expired', handleAuthExpired);
  window.removeEventListener('auth-updated', checkAuth);
});

const toggleTheme = () => {
  isDark.value = !isDark.value;
};

const toggleSidebar = () => {
  isSidebarExpanded.value = !isSidebarExpanded.value;
};

const handleAuthAction = () => {
  if (currentUser.value.isLoggedIn) {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    checkAuth();
    showToast('已退出当前账号', 'info');
    showLoginModal.value = true;
  } else {
    showLoginModal.value = true;
  }
};

const isActiveRoute = (itemPath) => {
  return route.path.startsWith(itemPath);
};
</script>

<template>
  <div class="app-container" :class="{ 'dark': isDark }">
    <ToastManager />
    <ConfirmModal />
    <LoginModal :isVisible="showLoginModal" @close="showLoginModal = false" @loginSuccess="checkAuth" />

    <aside class="sidebar" :class="{ 'expanded': isSidebarExpanded, 'collapsed': !isSidebarExpanded }">
      <div class="sidebar-top">
        <div class="top-controls">
          <button class="nav-toggle" @click="toggleSidebar">
            {{ isSidebarExpanded ? '← 收起' : '☰ 菜单' }}
          </button>
          <button v-if="isSidebarExpanded" class="theme-toggle" @click="toggleTheme">
            {{ isDark ? '☀ 日间' : '☾ 夜间' }}
          </button>
        </div>

        <div v-if="isSidebarExpanded" class="profile-link" @click="router.push('/dashboard')">
          <img :src="logoImg" alt="PaperCompare" class="avatar-small" />
          <div class="brand-meta">
            <span class="profile-name">PaperCompare</span>
            <span class="brand-tag">Paper Comparison Hub</span>
          </div>
        </div>
      </div>

      <nav v-if="isSidebarExpanded" class="sidebar-middle">
        <ul class="nav-links">
          <li
            v-for="item in navItems"
            :key="item.path"
            :class="{ active: isActiveRoute(item.path) }"
            @click="router.push(item.path)"
          >
            <span class="nav-text">{{ item.name }}</span>
            <span class="nav-sub">{{ item.sub }}</span>
          </li>
        </ul>
      </nav>

      <div v-if="isSidebarExpanded" class="sidebar-bottom">
        <div
          class="app-center-link"
          :class="{ active: route.path.startsWith('/comparisons/80001') }"
          @click="router.push('/comparisons/80001')"
        >
          <span class="nav-text">★ 联邦学习对比工作台</span>
        </div>

        <div class="user-info">
          <div class="avatar-text">
            {{ currentUser.nickname.charAt(0).toUpperCase() }}
          </div>
          <div class="user-details">
            <span class="user-name">
              {{ currentUser.nickname }}
            </span>
            <div class="user-links">
              <span class="user-action" @click="handleAuthAction">
                {{ currentUser.isLoggedIn ? '切换账号' : '登录' }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.sidebar {
  padding: 22px 0;
  justify-content: space-between;
}
.sidebar-top {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 0 20px;
}
.top-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}
.nav-toggle {
  font-weight: 700;
  letter-spacing: 0.8px;
  font-size: 13px;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--surface-bg);
}
.theme-toggle {
  font-size: 13px;
  font-weight: 600;
  opacity: 0.75;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--surface-bg);
}
.theme-toggle:hover {
  opacity: 1;
}
.profile-link {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  padding: 10px;
  border-radius: 14px;
  transition: background-color 0.2s;
  border: 1px solid var(--border-color);
}
.profile-link:hover {
  background-color: var(--surface-hover);
}
.avatar-small {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  object-fit: contain;
}
.brand-meta {
  display: flex;
  flex-direction: column;
}
.profile-name {
  font-weight: 800;
  font-size: 16px;
  letter-spacing: -0.3px;
}
.brand-tag {
  font-size: 11px;
  opacity: 0.55;
  font-weight: 600;
}
.sidebar-middle {
  padding: 0 20px;
  flex: 1;
  margin-top: 24px;
  overflow-y: auto;
}
.nav-links {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.nav-links li {
  display: flex;
  flex-direction: column;
  padding: 10px 14px;
  border-radius: 14px;
  opacity: 0.68;
  cursor: pointer;
  transition: all 0.2s ease;
}
.nav-links li .nav-text {
  font-size: 17px;
  font-weight: 800;
  letter-spacing: -0.3px;
}
.nav-links li .nav-sub {
  font-size: 11px;
  opacity: 0.65;
  text-transform: uppercase;
  letter-spacing: 0.8px;
}
.nav-links li:hover,
.nav-links li.active {
  opacity: 1;
  background: var(--surface-hover);
  transform: translateX(6px);
}
.sidebar-bottom {
  padding: 0 20px;
}
.app-center-link {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 11px 14px;
  border-radius: 12px;
  cursor: pointer;
  font-weight: 700;
  font-size: 14px;
  transition: all 0.2s;
  margin-bottom: 14px;
  background: var(--surface-bg);
  border: 1px solid var(--border-color);
}
.app-center-link:hover,
.app-center-link.active {
  background: var(--text-color);
  color: var(--bg-color);
}
.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}
.avatar-text {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: var(--text-color);
  color: var(--bg-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 14px;
}
.user-details {
  display: flex;
  flex-direction: column;
}
.user-name {
  font-weight: 700;
  font-size: 14px;
}
.user-links {
  display: flex;
  gap: 6px;
  font-size: 12px;
  opacity: 0.65;
}
.user-action {
  cursor: pointer;
}
.user-action:hover {
  opacity: 1;
  text-decoration: underline;
}
</style>
