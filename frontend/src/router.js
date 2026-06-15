import { createRouter, createWebHistory } from 'vue-router'
import { session } from './api'
import LoginView from './views/LoginView.vue'
import RegisterView from './views/RegisterView.vue'
import MetalListView from './views/MetalListView.vue'
import MetalDetailView from './views/MetalDetailView.vue'
import MetalFormView from './views/MetalFormView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/metals' },
    { path: '/login', component: LoginView, meta: { guest: true } },
    { path: '/register', component: RegisterView, meta: { guest: true } },
    { path: '/metals', component: MetalListView, meta: { auth: true } },
    { path: '/metals/new', component: MetalFormView, meta: { auth: true } },
    { path: '/metals/:id', component: MetalDetailView, meta: { auth: true } },
    { path: '/metals/:id/edit', component: MetalFormView, meta: { auth: true } }
  ]
})

router.beforeEach(to => {
  if (to.meta.auth && !session.authenticated) return '/login'
  if (to.meta.guest && session.authenticated) return '/metals'
})

export default router

