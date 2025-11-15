<script setup lang="ts">
import { onMounted, ref, computed } from 'vue';
import http from '@/services/http';
import type { Account, ApiResponse, Article } from '@/types/api';

const accounts = ref<Account[]>([]);
const selected = ref('');
const articles = ref<Article[]>([]);
const loading = ref(false);
const error = ref('');

const loadAccounts = async () => {
  const res = await http.get<ApiResponse<{ accounts: Account[] }>>('/api/accounts');
  if (res.data.success) {
    accounts.value = res.data.data.accounts;
  }
};

const loadArticles = async () => {
  if (!selected.value) return;
  loading.value = true;
  error.value = '';
  try {
    const res = await http.get<ApiResponse<{ account: Account; articles: Article[] }>>(
      `/api/accounts/${selected.value}/articles`,
    );
    if (res.data.success) {
      articles.value = res.data.data.articles;
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
};

const feedURL = computed(() =>
  selected.value ? `${window.location.origin}/feed/${selected.value}` : '',
);

onMounted(loadAccounts);
</script>

<template>
  <div class="card">
    <div class="toolbar">
      <select v-model="selected" class="input">
        <option value="">请选择公众号</option>
        <option v-for="account in accounts" :key="account.id" :value="account.id">
          {{ account.name }}
        </option>
      </select>
      <button class="btn btn-primary" :disabled="!selected" @click="loadArticles">查看文章</button>
      <a v-if="feedURL" class="feed-link" :href="feedURL" target="_blank" rel="noreferrer">
        RSS 订阅
      </a>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="loading">加载中...</p>
    <ul v-else class="article-list">
      <li v-for="article in articles" :key="article.id">
        <h3>
          <a :href="article.raw_url" target="_blank" rel="noreferrer">{{ article.title }}</a>
        </h3>
        <p class="meta">
          发布时间：{{ new Date(article.published_at).toLocaleString() }}
        </p>
        <p class="summary">{{ article.summary }}</p>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 0.8rem;
  margin-bottom: 1rem;
}

.feed-link {
  color: #2563eb;
}

.error {
  color: #dc2626;
}

.article-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.article-list li {
  border-bottom: 1px solid #e2e8f0;
  padding-bottom: 1rem;
}

.meta {
  color: #94a3b8;
  font-size: 0.9rem;
}

.summary {
  color: #475569;
}
</style>
