<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-brand">EINO 工作任务</div>
      <h1>登录</h1>
      <p class="auth-hint">使用账号密码登录后继续管理任务</p>
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="用户名">
          <el-input v-model="form.username" autocomplete="username" placeholder="用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            autocomplete="current-password"
            placeholder="密码"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" style="margin-bottom: 12px" />
        <el-button type="primary" class="auth-submit" :loading="loading" @click="submit">登录</el-button>
      </el-form>
      <div class="auth-footer">
        还没有账号？
        <router-link to="/register">注册</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login } from '../api'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const error = ref('')
const form = reactive({ username: '', password: '' })

const submit = async () => {
  error.value = ''
  if (!form.username || !form.password) {
    error.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  try {
    await login({ username: form.username.trim(), password: form.password })
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect || '/')
  } catch (e) {
    error.value = e.response?.data?.error || e.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>
