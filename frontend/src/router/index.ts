import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: () => import('../views/HomeView.vue') },
  { path: '/g/:groupId', name: 'group', component: () => import('../views/GroupView.vue'), props: true },
  { path: '/c/:cardId', name: 'card', component: () => import('../views/CardView.vue'), props: true },
  { path: '/chat', name: 'chat', component: () => import('../views/ChatView.vue') },
  { path: '/search', name: 'search', component: () => import('../views/SearchView.vue') },
  { path: '/trash', name: 'trash', component: () => import('../views/TrashView.vue') },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});
