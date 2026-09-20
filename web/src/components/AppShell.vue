<template>
  <div class="app-shell">
    <aside class="side">
      <div class="brand">EINO 工作任务</div>
      <div class="brand-sub">长任务 Harness</div>
      <nav class="side-nav">
        <router-link to="/" :class="{ 'is-active': route.path === '/' }">任务</router-link>
        <router-link to="/projects" :class="{ 'is-active': route.path.startsWith('/projects') }">项目</router-link>
        <router-link to="/schedules" :class="{ 'is-active': route.path.startsWith('/schedules') }">定时</router-link>
        <router-link to="/connectors" :class="{ 'is-active': route.path.startsWith('/connectors') }">连接器</router-link>
        <router-link to="/audit" :class="{ 'is-active': route.path.startsWith('/audit') }">审计</router-link>
        <router-link to="/eval" :class="{ 'is-active': route.path.startsWith('/eval') }">评估</router-link>
      </nav>
      <slot name="side" />
    </aside>
    <main class="main">
      <div class="app-userbar">
        <span class="app-userbar-name" :title="user?.username || ''">{{ accountName }}</span>
        <el-button size="small" @click="logout">退出登录</el-button>
      </div>
      <slot />
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getMe } from '../api'
import { accountLabel, clearSession, currentUser, getToken, setSession } from '../auth'

const route = useRoute()
const router = useRouter()
const user = currentUser
const accountName = computed(() => accountLabel(user.value))

onMounted(async () => {
  const token = getToken()
  if (!token) return
  try {
    const me = await getMe()
    setSession(token, me)
  } catch {
    // 保留本地会话，接口失败时仍显示已缓存的名称
  }
})

const logout = () => {
  clearSession()
  router.replace('/login')
}
</script>
