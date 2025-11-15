<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue';
import http from '@/services/http';
import type { ApiResponse, WechatSession } from '@/types/api';

const sessions = ref<WechatSession[]>([]);
const loading = ref(false);
const error = ref('');
let timer: number | undefined;

const loadSessions = async () => {
  loading.value = true;
  error.value = '';
  try {
    const res = await http.get<ApiResponse<{ sessions: WechatSession[] }>>('/api/wechat/sessions');
    if (res.data.success) {
      sessions.value = res.data.data.sessions;
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
};

const createSession = async () => {
  const res = await http.post<ApiResponse<{ session: WechatSession }>>('/api/wechat/sessions', {});
  if (res.data.success) {
    sessions.value.unshift(res.data.data.session);
  }
};

const startAutoRefresh = () => {
  timer = window.setInterval(loadSessions, 5000);
};

onMounted(() => {
  loadSessions();
  startAutoRefresh();
});

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer);
  }
});
</script>

<template>
  <div class="card">
    <div class="list-header">
      <h2>微信会话</h2>
      <div class="actions">
        <button class="btn" @click="loadSessions" :disabled="loading">刷新</button>
        <button class="btn btn-primary" @click="createSession">生成二维码</button>
      </div>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <table>
      <thead>
        <tr>
          <th>ID</th>
          <th>状态</th>
          <th>二维码</th>
          <th>创建时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="session in sessions" :key="session.id">
          <td>{{ session.id }}</td>
          <td>
            <span :class="['tag', session.status]">{{ session.status }}</span>
          </td>
          <td>
            <img v-if="session.qr_code" :src="session.qr_code" alt="二维码" class="qr" />
          </td>
          <td>{{ new Date(session.created_at).toLocaleString() }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.actions {
  display: flex;
  gap: 0.5rem;
}

.error {
  color: #dc2626;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 0.55rem;
  border-bottom: 1px solid #e2e8f0;
  text-align: left;
}

.qr {
  width: 120px;
  height: 120px;
  object-fit: contain;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
}

.tag {
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  text-transform: capitalize;
}

.tag.active {
  background: #dcfce7;
  color: #166534;
}

.tag.pending,
.tag.scanning {
  background: #fef3c7;
  color: #92400e;
}

.tag.expired {
  background: #fee2e2;
  color: #b91c1c;
}
</style>
