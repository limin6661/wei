<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink, RouterView, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();
const router = useRouter();

const navItems = [
  { name: '总览', path: '/' },
  { name: '公众号', path: '/accounts' },
  { name: '文章', path: '/articles' },
  { name: '任务', path: '/tasks' },
  { name: '微信会话', path: '/sessions' },
];

const userName = computed(() => auth.user?.username ?? '管理员');

const handleLogout = async () => {
  await auth.logout();
  router.push('/login');
};
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">Wechat2RSS</div>
      <nav>
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-link"
          active-class="active"
        >
          {{ item.name }}
        </RouterLink>
      </nav>
    </aside>

    <main class="content">
      <header class="topbar">
        <div class="user-info">
          <span>{{ userName }}</span>
          <span v-if="auth.needsReset" class="tag warning">需改密码</span>
        </div>
        <button class="btn btn-primary" @click="handleLogout">退出</button>
      </header>

      <section class="view">
        <RouterView />
      </section>
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  min-height: 100vh;
}

.sidebar {
  background: #111827;
  color: white;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.brand {
  font-weight: 600;
  font-size: 1.2rem;
}

.nav-link {
  display: block;
  padding: 0.6rem 0.8rem;
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.85);
  margin-bottom: 0.2rem;
  transition: background 0.2s ease;
}

.nav-link:hover {
  background: rgba(255, 255, 255, 0.12);
}

.nav-link.active {
  background: #2563eb;
  color: white;
}

.content {
  background: #f8fafc;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #475569;
  font-weight: 500;
}

.tag {
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  font-size: 0.75rem;
}

.tag.warning {
  background: #fef3c7;
  color: #92400e;
}

.view {
  flex: 1;
}
</style>
