import { createRouter, createWebHistory } from 'vue-router'
import Home from './views/Home.vue'
import Projects from './views/Projects.vue'
import Schedules from './views/Schedules.vue'
import Connectors from './views/Connectors.vue'
import Audit from './views/Audit.vue'
import Eval from './views/Eval.vue'
import Login from './views/Login.vue'
import Register from './views/Register.vue'
import { isLoggedIn } from './auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login, meta: { public: true } },
    { path: '/register', component: Register, meta: { public: true } },
    { path: '/', component: Home },
    { path: '/projects', component: Projects },
    { path: '/schedules', component: Schedules },
    { path: '/connectors', component: Connectors },
    { path: '/audit', component: Audit },
    { path: '/eval', component: Eval },
  ],
})

router.beforeEach((to) => {
  if (to.meta.public) {
    if (isLoggedIn() && (to.path === '/login' || to.path === '/register')) {
      return { path: '/' }
    }
    return true
  }
  if (!isLoggedIn()) {
    return {
      path: '/login',
      query: { redirect: to.fullPath },
    }
  }
  return true
})

export default router
