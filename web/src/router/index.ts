import { createRouter, createWebHistory } from 'vue-router';
import LoginView from '@/views/LoginView.vue';
import DashboardLayout from '@/views/DashboardLayout.vue';
import OverviewView from '@/views/OverviewView.vue';
import AccountsView from '@/views/AccountsView.vue';
import TasksView from '@/views/TasksView.vue';
import SessionsView from '@/views/SessionsView.vue';
import ArticlesView from '@/views/ArticlesView.vue';
import { useAuthStore } from '@/stores/auth';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
    },
    {
      path: '/',
      component: DashboardLayout,
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'overview',
          component: OverviewView,
        },
        {
          path: 'accounts',
          name: 'accounts',
          component: AccountsView,
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: TasksView,
        },
        {
          path: 'articles',
          name: 'articles',
          component: ArticlesView,
        },
        {
          path: 'sessions',
          name: 'sessions',
          component: SessionsView,
        },
      ],
    },
  ],
});

router.beforeEach(async (to, _from, next) => {
  const auth = useAuthStore();

  if (!auth.initialized) {
    await auth.fetchMe().catch(() => undefined);
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  if (to.name === 'login' && auth.isAuthenticated) {
    next({ name: 'overview' });
    return;
  }

  next();
});

export default router;
