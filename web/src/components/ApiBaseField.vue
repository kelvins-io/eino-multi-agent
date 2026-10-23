<template>
  <el-form-item v-if="visible" label="API 地址">
    <el-input
      v-model="draft"
      placeholder="http://127.0.0.1:8180"
      clearable
      @change="persist"
      @blur="persist"
    />
    <p class="api-base-hint">桌面端连接的后端地址，留空则使用默认值</p>
  </el-form-item>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { getStoredApiBase, isDesktopShell, resolveApiOrigin, setStoredApiBase } from '../apiBase'

const visible = isDesktopShell() || Boolean(String(import.meta.env.VITE_API_BASE || '').trim())
const draft = ref('')

onMounted(() => {
  draft.value = getStoredApiBase() || resolveApiOrigin() || 'http://127.0.0.1:8180'
})

const persist = () => {
  setStoredApiBase(draft.value)
}
</script>
