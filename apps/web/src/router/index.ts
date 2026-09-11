import { createRouter, createWebHistory } from 'vue-router'
import IndexPage from '../pages/index/index.vue'
import LoginPage from '../pages/login/index.vue'
import CapturePage from '../pages/meal/capture.vue'
import AnalyzingPage from '../pages/meal/analyzing.vue'
import ConfirmPage from '../pages/meal/confirm.vue'
import ManualPage from '../pages/meal/manual.vue'
import GoalPage from '../pages/onboarding/goal.vue'
import ProfilePage from '../pages/onboarding/profile.vue'
import ResultPage from '../pages/onboarding/result.vue'
import TodayPage from '../pages/today/index.vue'
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
    { path: '/today', component: TodayPage, meta: { requiresAuth: true } },
    { path: '/meals/new', component: CapturePage, meta: { requiresAuth: true } },
    { path: '/meals/analyzing', component: AnalyzingPage, meta: { requiresAuth: true } },
    { path: '/meals/confirm', component: ConfirmPage, meta: { requiresAuth: true } },
    { path: '/meals/manual', component: ManualPage, meta: { requiresAuth: true } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach((to) => guardRoute(to.path, Boolean(to.meta.requiresAuth), sessionStore))
