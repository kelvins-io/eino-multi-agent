<template>
  <AppShell>
    <template #side>
      <el-button type="primary" class="side-action" @click="resetComposer">新建连接器</el-button>
      <div class="task-list">
        <div
          v-for="item in connectors"
          :key="item.id"
          class="task-item"
          :class="{ active: currentId === item.id }"
          @click="currentId = item.id"
        >
          <div class="title">{{ item.name }}</div>
          <div class="meta">
            <el-tag size="small" :type="item.enabled ? 'success' : 'info'">{{ kindText(item.kind) }}</el-tag>
            <span>{{ item.enabled ? '启用' : '停用' }}</span>
          </div>
        </div>
        <div v-if="!connectors.length" class="empty">还没有连接器</div>
      </div>
    </template>

    <div class="page-body">
      <section class="panel page-panel">
        <div v-if="!current">
          <h3>任务完成后投递</h3>
          <p class="muted">Webhook 会 POST 任务结果；本地目录会把产物复制到 workspace/exports。</p>
          <el-form label-position="top">
            <el-form-item label="名称">
              <el-input v-model="form.name" placeholder="例如：周报推送" />
            </el-form-item>
            <el-form-item label="类型">
              <el-radio-group v-model="form.kind">
                <el-radio-button label="webhook">Webhook</el-radio-button>
                <el-radio-button label="local_dir">本地目录</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.kind === 'webhook'" label="URL">
              <el-input v-model="form.url" placeholder="本机可填 /api/v1/hooks/echo 的完整地址" />
            </el-form-item>
            <el-form-item v-if="form.kind === 'webhook'" label="密钥（可选）">
              <el-input v-model="form.secret" placeholder="请求头 X-Harness-Secret" />
            </el-form-item>
            <el-form-item v-else label="导出目录名">
              <el-input v-model="form.path" placeholder="默认使用连接器 ID，实际路径 workspace/exports/{name}" />
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="creating" :disabled="!form.name" @click="submit">保存</el-button>
        </div>
        <div v-else>
          <div style="display: flex; justify-content: space-between; align-items: center; gap: 12px">
            <div>
              <h3 style="margin: 0 0 6px">{{ current.name }}</h3>
              <el-tag>{{ kindText(current.kind) }}</el-tag>
              <el-tag size="small" :type="current.enabled ? 'success' : 'info'" style="margin-left: 6px">
                {{ current.enabled ? '启用' : '停用' }}
              </el-tag>
            </div>
            <div>
              <el-button @click="onToggle">{{ current.enabled ? '停用' : '启用' }}</el-button>
              <el-button type="primary" :loading="testing" @click="onTest">测试</el-button>
            </div>
          </div>
          <p class="muted" style="white-space: pre-wrap">{{ configText(current) }}</p>
        </div>
      </section>
    </div>
  </AppShell>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/AppShell.vue'
import { createConnector, getMeta, listConnectors, testConnector, toggleConnector } from '../api'

const meta = ref({ llm: {} })
const connectors = ref([])
const currentId = ref('')
const creating = ref(false)
const testing = ref(false)
const form = reactive({
  name: '',
  kind: 'webhook',
  url: 'http://127.0.0.1:8180/api/v1/hooks/echo',
  secret: '',
  path: '',
})

const current = computed(() => connectors.value.find((c) => c.id === currentId.value) || null)

const kindText = (kind) => (kind === 'local_dir' ? '本地目录' : 'Webhook')

const configText = (item) => {
  if (!item?.config) return ''
  if (item.kind === 'webhook') return item.config.url || ''
  return `workspace/exports/${item.config.path || item.id}`
}

const resetComposer = () => {
  currentId.value = ''
}

const refresh = async () => {
  connectors.value = await listConnectors()
}

const submit = async () => {
  creating.value = true
  try {
    const config = form.kind === 'webhook' ? { url: form.url, secret: form.secret } : { path: form.path }
    const item = await createConnector({ name: form.name, kind: form.kind, config })
    ElMessage.success('连接器已保存')
    form.name = ''
    form.url = ''
    form.secret = ''
    form.path = ''
    await refresh()
    currentId.value = item.id
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    creating.value = false
  }
}

const onToggle = async () => {
  try {
    const item = await toggleConnector(currentId.value)
    ElMessage.success(item.enabled ? '已启用' : '已停用')
    await refresh()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

const onTest = async () => {
  testing.value = true
  try {
    await testConnector(currentId.value)
    ElMessage.success('测试成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    testing.value = false
  }
}

onMounted(async () => {
  try {
    meta.value = await getMeta()
    await refresh()
  } catch (err) {
    ElMessage.error('无法连接后端，请先启动 API 服务')
  }
})
</script>
