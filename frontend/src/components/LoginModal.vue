<script setup>
import { ref } from 'vue';
import api, { unwrap } from '../api';
import { showToast } from '../utils/toast';

const props = defineProps({
  isVisible: Boolean
});
const emit = defineEmits(['close', 'loginSuccess']);

const mode = ref('login'); // 'login' | 'register'
const email = ref('demo@papercompare.org');
const password = ref('123456');
const nickname = ref('');
const loading = ref(false);

const saveAuthData = (data) => {
  localStorage.setItem('access_token', data.access_token);
  localStorage.setItem('refresh_token', data.refresh_token || '');
  if (data.user) {
    localStorage.setItem('user_id', String(data.user.id));
    localStorage.setItem('user_email', data.user.email);
    localStorage.setItem('user_nickname', data.user.nickname);
  }
  window.dispatchEvent(new Event('auth-updated'));
  emit('loginSuccess', data.user);
  emit('close');
};

const handleSubmit = async () => {
  if (!email.value || !password.value) {
    showToast('请填写邮箱和密码', 'error');
    return;
  }
  loading.value = true;
  try {
    if (mode.value === 'login') {
      const res = await api.post('/api/auth/login', {
        email: email.value,
        password: password.value
      });
      const data = unwrap(res);
      showToast(`欢迎回来，${data.user?.nickname || '研究员'}`, 'info');
      saveAuthData(data);
    } else {
      const res = await api.post('/api/auth/register', {
        email: email.value,
        password: password.value,
        nickname: nickname.value || email.value.split('@')[0]
      });
      const data = unwrap(res);
      showToast('注册并登录成功', 'info');
      saveAuthData(data);
    }
  } catch (err) {
    showToast(err.message || '认证失败', 'error');
  } finally {
    loading.value = false;
  }
};

const fillDemoAccount = () => {
  mode.value = 'login';
  email.value = 'demo@papercompare.org';
  password.value = '123456';
  handleSubmit();
};
</script>

<template>
  <div v-if="isVisible" class="modal-backdrop" @click="emit('close')">
    <div class="modal-card login-box" @click.stop>
      <div class="login-header">
        <h2>{{ mode === 'login' ? '登录 PaperCompare' : '注册科研账号' }}</h2>
        <p class="page-subtitle">分布式科研文献管理、结构化提取与多论文横向对比平台</p>
      </div>

      <form class="login-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label>学术邮箱 (Email)</label>
          <input v-model="email" type="email" placeholder="demo@papercompare.org" required />
        </div>

        <div v-if="mode === 'register'" class="form-group">
          <label>学者昵称 (Nickname)</label>
          <input v-model="nickname" type="text" placeholder="Dr. GouTou" />
        </div>

        <div class="form-group">
          <label>登录密码 (Password)</label>
          <input v-model="password" type="password" placeholder="至少 6 位密码" required />
        </div>

        <button type="submit" class="btn-primary submit-btn" :disabled="loading">
          {{ loading ? '正在验证...' : (mode === 'login' ? '立即进入工作台' : '创建账号并登录') }}
        </button>
      </form>

      <div class="demo-shortcut">
        <button type="button" class="btn-outline full-btn" @click="fillDemoAccount">
          ⚡ 一键使用预置演示账号登录 (Dr. GouTou · 含 5 篇联邦学习论文)
        </button>
      </div>

      <div class="switch-row">
        <span v-if="mode === 'login'">
          还没有账号？ <a href="#" @click.prevent="mode = 'register'">立即注册</a>
        </span>
        <span v-else>
          已有账号？ <a href="#" @click.prevent="mode = 'login'">返回登录</a>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-box {
  max-width: 440px;
}
.login-header {
  margin-bottom: 24px;
}
.login-header h2 {
  font-size: 1.75rem;
  font-weight: 800;
  letter-spacing: -0.5px;
}
.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-group label {
  font-size: 0.85rem;
  font-weight: 700;
  opacity: 0.8;
}
.submit-btn {
  width: 100%;
  justify-content: center;
  padding: 13px;
  margin-top: 8px;
}
.demo-shortcut {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}
.full-btn {
  width: 100%;
  justify-content: center;
  font-size: 0.84rem;
}
.switch-row {
  margin-top: 18px;
  text-align: center;
  font-size: 0.9rem;
  opacity: 0.8;
}
.switch-row a {
  font-weight: 700;
  text-decoration: underline;
}
</style>

