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
      <slot />
    </main>
  </div>
</template>

<script setup>
import { useRoute } from 'vue-router'

const route = useRoute()
</script>
