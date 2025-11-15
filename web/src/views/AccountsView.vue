<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import http from '@/services/http';
import type {
  Account,
  ApiResponse,
  WechatSession,
  WechatSearchResult,
} from '@/types/api';

const accounts = ref<Account[]>([]);
const sessions = ref<WechatSession[]>([]);
const loading = ref(false);
const creating = ref(false);
const error = ref('');
const searchError = ref('');

const form = reactive({
  name: '',
  wechat_id: '',
  biz_id: '',
  alias: '',
  session_id: '',
});

const searchQuery = ref('');
const searchResults = ref<WechatSearchResult[]>([]);

const loadAccounts = async () => {
  loading.value = true;
  error.value = '';
  try {
    const res = await http.get<ApiResponse<{ accounts: Account[] }>>('/api/accounts');
    if (res.data.success) {
      accounts.value = res.data.data.accounts;
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
};

const loadSessions = async () => {
  const res = await http.get<ApiResponse<{ sessions: WechatSession[] }>>('/api/wechat/sessions');
  if (res.data.success) {
    sessions.value = res.data.data.sessions;
  }
};

const createAccount = async () => {
  creating.value = true;
  error.value = '';
  try {
    const payload = {
      name: form.name,
      wechat_id: form.wechat_id,
      alias: form.alias,
      biz_id: form.biz_id,
      session_id: form.session_id ? Number(form.session_id) : undefined,
    };
    const res = await http.post<ApiResponse<{ account: Account }>>('/api/accounts', payload);
    if (res.data.success) {
      accounts.value.unshift(res.data.data.account);
      form.name = '';
      form.wechat_id = '';
      form.biz_id = '';
      form.alias = '';
      form.session_id = '';
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '创建失败';
  } finally {
    creating.value = false;
  }
};

const searchWechat = async () => {
  searchError.value = '';
  if (!form.session_id) {
    searchError.value = '请先选择可用的会话';
    return;
  }
  if (!searchQuery.value) {
    searchError.value = '请输入公众号名称';
    return;
  }
  try {
    const res = await http.get<ApiResponse<{ results: WechatSearchResult[] }>>(
      '/api/wechat/search',
      {
        params: {
          session_id: form.session_id,
          query: searchQuery.value,
        },
      },
    );
    if (res.data.success) {
      searchResults.value = res.data.data.results;
    }
  } catch (err) {
    searchError.value = err instanceof Error ? err.message : '搜索失败';
  }
};

const useSearchResult = (result: WechatSearchResult) => {
  form.name = result.nickname;
  form.wechat_id = result.alias || result.nickname;
  form.biz_id = result.fakeid;
};

onMounted(() => {
  loadAccounts();
  loadSessions();
});
</script>

<template>
  <div class="grid">
    <div class="card">
      <h2>添加公众号</h2>
      <form class="form" @submit.prevent="createAccount">
        <label>
          名称
          <input v-model="form.name" class="input" required />
        </label>
        <label>
          原始 ID
          <input v-model="form.wechat_id" class="input" required />
        </label>
        <label>
          Biz/FakeID
          <input v-model="form.biz_id" class="input" placeholder="可通过下方搜索获取" />
        </label>
        <label>
          绑定会话
          <select v-model="form.session_id" class="input">
            <option value="">请选择</option>
            <option
              v-for="session in sessions"
              :key="session.id"
              :value="session.id"
              :disabled="session.status !== 'active'"
            >
              #{{ session.id }} - {{ session.status }}
            </option>
          </select>
        </label>
        <label>
          备注
          <input v-model="form.alias" class="input" />
        </label>
        <p v-if="error" class="error">{{ error }}</p>
        <button class="btn btn-primary" :disabled="creating">
          {{ creating ? '提交中...' : '保存' }}
        </button>
      </form>

      <div class="search-panel">
        <h3>快速搜索 BizID</h3>
        <div class="search-row">
          <input v-model="searchQuery" class="input" placeholder="输入公众号名称" />
          <button class="btn" type="button" @click="searchWechat">搜索</button>
        </div>
        <p v-if="searchError" class="error">{{ searchError }}</p>
        <ul v-if="searchResults.length" class="results">
          <li v-for="item in searchResults" :key="item.fakeid">
            <div>
              <strong>{{ item.nickname }}</strong> <span>{{ item.alias }}</span>
              <p>FakeID: {{ item.fakeid }}</p>
            </div>
            <button class="btn" type="button" @click="useSearchResult(item)">使用</button>
          </li>
        </ul>
      </div>
    </div>

    <div class="card">
      <div class="list-header">
        <h2>公众号列表</h2>
        <button class="btn" @click="loadAccounts" :disabled="loading">刷新</button>
      </div>
      <p v-if="loading">加载中...</p>
      <table v-else>
        <thead>
          <tr>
            <th>ID</th>
            <th>名称</th>
            <th>原始 ID</th>
            <th>BizID</th>
            <th>会话</th>
            <th>状态</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="account in accounts" :key="account.id">
            <td>{{ account.id }}</td>
            <td>{{ account.name }}</td>
            <td>{{ account.wechat_id }}</td>
            <td>{{ account.biz_id }}</td>
            <td>{{ account.session_id ?? '未绑定' }}</td>
            <td>
              <span :class="['tag', account.status === 'active' ? 'success' : 'warning']">
                {{ account.status }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
  gap: 1.5rem;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
}

label {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  font-size: 0.95rem;
}

.error {
  color: #dc2626;
}

.search-panel {
  margin-top: 1.5rem;
}

.search-row {
  display: flex;
  gap: 0.8rem;
  margin-bottom: 0.5rem;
}

.results {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.results li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 0.6rem;
}

table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 1rem;
}

th,
td {
  padding: 0.5rem;
  border-bottom: 1px solid #e2e8f0;
  text-align: left;
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.tag {
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  text-transform: capitalize;
}

.tag.success {
  background: #dcfce7;
  color: #166534;
}

.tag.warning {
  background: #fee2e2;
  color: #991b1b;
}
</style>
