import { createRouter, createWebHistory } from 'vue-router';

import DashboardView from '../views/DashboardView.vue';
import ComparisonListView from '../views/ComparisonListView.vue';
import ComparisonWorkspaceView from '../views/ComparisonWorkspaceView.vue';

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', name: 'dashboard', component: DashboardView },
  { path: '/comparisons', name: 'comparisons', component: ComparisonListView },
  { path: '/comparisons/:id', name: 'comparison-workspace', component: ComparisonWorkspaceView },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
