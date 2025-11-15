<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import http from '@/services/http';
import type { ApiResponse, Task, TaskLog } from '@/types/api';

const tasks = ref<Task[]>([]);
const logs = ref<TaskLog[]>([]);
const logsVisible = ref(false);
const currentTask = ref<Task | null>(null);
const loading = ref(false);
const triggerState = reactive({
  accountId: '',
  running: false,
});
const message = ref('');

const loadTasks = async () => {
  loading.value = true;
  try {
    const res = await http.get<ApiResponse<{ tasks: Task[] }>>('/api/tasks');
    if (res.data.success) {
      tasks.value = res.data.data.tasks;
    }
  } finally {
    loading.value = false;
  }
};

const triggerTask = async () => {
  if (!triggerState.accountId) return;
  triggerState.running = true;
  message.value = '';
  try {
    const res = await http.post<ApiResponse<{ task: Task }>>(
      `/api/accounts/${triggerState.accountId}/tasks`,
      {},
    );
    if (res.data.success) {
      tasks.value.unshift(res.data.data.task);
      message.value = `已创建任务 #${res.data.data.task.id}`;
    }
  } catch (err) {
    message.value = err instanceof Error ? err.message : '触发失败';
  } finally {
    triggerState.running = false;
  }
};

const openLogs = async (task: Task) => {
  currentTask.value = task;
  const res = await http.get<ApiResponse<{ logs: TaskLog[] }>>(`/api/tasks/${task.id}/logs`);
  logs.value = res.data.data.logs;
  logsVisible.value = true;
};

onMounted(loadTasks);
</script>

<template>
  <div class="card">
    <div class="list-header">
      <h2>抓取任务</h2>
      <button class="btn" @click="loadTasks" :disabled="loading">刷新</button>
    </div>
    <div class="trigger">
      <input
        v-model="triggerState.accountId"
        class="input"
        placeholder="输入公众号 ID 触发一次抓取"
      />
      <button class="btn btn-primary" :disabled="triggerState.running" @click="triggerTask">
        创建任务
      </button>
    </div>
    <p v-if="message">{{ message }}</p>
    <table>
      <thead>
        <tr>
          <th>ID</th>
          <th>公众号</th>
          <th>状态</th>
          <th>错误</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="task in tasks" :key="task.id">
          <td>{{ task.id }}</td>
          <td>{{ task.account?.name ?? task.account_id }}</td>
          <td>
            <span :class="['tag', task.status]">{{ task.status }}</span>
          </td>
          <td>
            <span class="error-text">{{ task.error }}</span>
          </td>
          <td>
            <button class="btn" @click="openLogs(task)">日志</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>

  <div v-if="logsVisible" class="modal-backdrop" @click.self="logsVisible = false">
    <div class="modal card">
      <div class="modal-header">
        <h3>任务日志 #{{ currentTask?.id }}</h3>
        <button class="btn" @click="logsVisible = false">关闭</button>
      </div>
      <ul>
        <li v-for="log in logs" :key="log.id">
          <span class="log-time">{{ new Date(log.created_at).toLocaleString() }}</span>
          <span :class="['log-level', log.level]">{{ log.level }}</span>
          {{ log.message }}
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.trigger {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1rem;
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

.tag {
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  text-transform: capitalize;
}

.tag.pending {
  background: #fef3c7;
  color: #92400e;
}

.tag.running {
  background: #dbeafe;
  color: #1d4ed8;
}

.tag.success {
  background: #dcfce7;
  color: #166534;
}

.tag.failed {
  background: #fee2e2;
  color: #b91c1c;
}

.error-text {
  color: #dc2626;
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.modal {
  width: min(520px, 90vw);
  max-height: 70vh;
  overflow: auto;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.log-time {
  font-size: 0.8rem;
  color: #94a3b8;
  margin-right: 0.5rem;
}

.log-level {
  font-size: 0.75rem;
  text-transform: uppercase;
  margin-right: 0.5rem;
}

.log-level.error {
  color: #dc2626;
}

.log-level.info {
  color: #2563eb;
}
</style>
