import { createRouter, createWebHistory } from 'vue-router'
import IndexPage from '../pages/index/index.vue'
import LoginPage from '../pages/login/index.vue'
import GoalPage from '../pages/onboarding/goal.vue'
import ProfilePage from '../pages/onboarding/profile.vue'
import ResultPage from '../pages/onboarding/result.vue'
import ProtectedPage from '../pages/protected/index.vue'
import WelcomePage from '../pages/welcome/index.vue'
import { sessionStore } from '../stores/session'
import { guardRoute } from './boot'

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: IndexPage },
    { path: '/welcome', component: WelcomePage },
    { path: '/login', component: LoginPage },
    { path: '/profile', component: ProfilePage, meta: { requiresAuth: true } },
    { path: '/goals', component: GoalPage, meta: { requiresAuth: true } },
    { path: '/goals/result', component: ResultPage, meta: { requiresAuth: true } },
    { path: '/today', component: ProtectedPage, props: { title: '今天', description: '登录状态已恢复。', allowLogout: true }, meta: { requiresAuth: true } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach((to) => guardRoute(to.path, Boolean(to.meta.requiresAuth), sessionStore))
