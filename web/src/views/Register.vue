<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-brand">EINO 工作任务</div>
      <h1>注册</h1>
      <p class="auth-hint">创建账号后即可使用工作台</p>
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="用户名">
          <el-input v-model="form.username" autocomplete="username" placeholder="字母数字 _ -，至少 3 位" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="form.display_name" autocomplete="nickname" placeholder="界面展示用昵称" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="至少 6 位"
          />
        </el-form-item>
        <el-form-item label="确认密码">
          <el-input
            v-model="form.confirm"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="再次输入密码"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" style="margin-bottom: 12px" />
        <el-button type="primary" class="auth-submit" :loading="loading" @click="submit">注册并登录</el-button>
      </el-form>
      <div class="auth-footer">
        已有账号？
        <router-link to="/login">登录</router-link>
      </div>
    </div>
    <div class="auth-contact">
      联系我们
      <a href="mailto:1225807604@qq.com">1225807604@qq.com</a>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../api'

const router = useRouter()
const loading = ref(false)
const error = ref('')
const form = reactive({ username: '', display_name: '', password: '', confirm: '' })

const submit = async () => {
  error.value = ''
  if (!form.username || !form.display_name || !form.password) {
    error.value = '请填写用户名、显示名和密码'
    return
  }
  if (form.password !== form.confirm) {
    error.value = '两次输入的密码不一致'
    return
  }
  loading.value = true
  try {
    await register({
      username: form.username.trim(),
      display_name: form.display_name.trim(),
      password: form.password,
    })
    await router.replace('/')
  } catch (e) {
    error.value = e.response?.data?.error || e.message || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>
