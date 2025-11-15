<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const form = reactive({
  username: 'admin',
  password: '',
});

const resetForm = reactive({
  old: '',
  fresh: '',
});

const loading = ref(false);
const error = ref('');
const resetError = ref('');

const redirectAfterLogin = () => {
  const redirect = (route.query.redirect as string) || '/';
  router.push(redirect);
};

const handleLogin = async () => {
  error.value = '';
  loading.value = true;
  try {
    const user = await auth.login({
      username: form.username,
      password: form.password,
    });

    if (!user.require_reset) {
      redirectAfterLogin();
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败';
  } finally {
    loading.value = false;
  }
};

const handleReset = async () => {
  resetError.value = '';
  loading.value = true;
  try {
    await auth.updatePassword(resetForm.old, resetForm.fresh);
    redirectAfterLogin();
  } catch (err) {
    resetError.value = err instanceof Error ? err.message : '修改失败';
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="login-page">
    <div class="card login-card">
      <h1>Wechat2RSS 控制台</h1>
      <p class="subtitle">登录后台，管理公众号抓取任务</p>

      <form class="form" @submit.prevent="handleLogin">
        <label>
          用户名
          <input v-model="form.username" class="input" placeholder="admin" />
        </label>
        <label>
          密码
          <input v-model="form.password" type="password" class="input" placeholder="请输入密码" />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button class="btn btn-primary" :disabled="loading">
          {{ auth.user?.require_reset ? '验证密码' : '登录' }}
        </button>
      </form>

      <div v-if="auth.user?.require_reset" class="reset-panel">
        <h3>首次登录需修改密码</h3>
        <form @submit.prevent="handleReset">
          <label>
            旧密码
            <input v-model="resetForm.old" type="password" class="input" />
          </label>
          <label>
            新密码（至少 8 位）
            <input v-model="resetForm.fresh" type="password" class="input" minlength="8" />
          </label>
          <p v-if="resetError" class="error">{{ resetError }}</p>
          <button class="btn btn-primary" :disabled="loading">保存新密码</button>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at top, #dbeafe, #e0e7ff, #f8fafc);
}

.login-card {
  width: min(420px, 90vw);
}

.subtitle {
  color: #64748b;
  margin-top: 0.25rem;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
  margin-top: 1.2rem;
}

label {
  display: flex;
  flex-direction: column;
  color: #0f172a;
  gap: 0.35rem;
  font-size: 0.95rem;
}

.error {
  color: #dc2626;
  font-size: 0.9rem;
}

.reset-panel {
  margin-top: 1.5rem;
  padding-top: 1.2rem;
  border-top: 1px solid #e2e8f0;
}
</style>
