import { createRouter, createWebHistory } from 'vue-router'
import Home from './views/Home.vue'
import Projects from './views/Projects.vue'
import Schedules from './views/Schedules.vue'
import Connectors from './views/Connectors.vue'
import Audit from './views/Audit.vue'
import Eval from './views/Eval.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/projects', component: Projects },
    { path: '/schedules', component: Schedules },
    { path: '/connectors', component: Connectors },
    { path: '/audit', component: Audit },
    { path: '/eval', component: Eval },
  ],
})
