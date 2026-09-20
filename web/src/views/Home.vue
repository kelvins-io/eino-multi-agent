<template>
  <div class="app-shell">
    <aside class="side">
      <div class="brand">EINO 工作任务</div>
      <div class="brand-sub">长任务 Harness · {{ meta.llm?.model || '未配置模型' }}</div>
      <el-button type="primary" class="side-action" @click="resetComposer">
        新建任务
      </el-button>
      <div class="task-list">
        <div
          v-for="item in tasks"
          :key="item.id"
          class="task-item"
          :class="{ active: currentId === item.id }"
          @click="selectTask(item.id)"
        >
          <div class="title">{{ item.title }}</div>
          <div class="meta">
            <el-tag size="small" :type="statusType(item.status)">{{ statusText(item.status) }}</el-tag>
            <span>{{ formatTime(item.created_at) }}</span>
          </div>
        </div>
        <div v-if="!tasks.length" class="empty">还没有任务</div>
      </div>
    </aside>

    <main class="main">
      <div class="topbar">
        <el-alert
          v-if="!meta.llm?.ready"
          title="尚未配置 LLM。设置 EINO_LLM_API_KEY 或 config.yaml 后即可真正跑任务。"
          type="warning"
          show-icon
          :closable="false"
        />
      </div>

      <div class="workspace">
        <section class="panel">
          <div v-if="!detail" class="composer">
            <h3>下达工作任务</h3>
            <el-input v-model="form.title" placeholder="标题（可空，默认截取目标）" />
            <el-input
              v-model="form.goal"
              type="textarea"
              :rows="8"
              placeholder="说明要完成什么、材料在哪、交付什么。例如：根据 input/sales.csv 生成本周销售周报，输出 Markdown。"
            />
            <el-form label-position="top">
              <el-form-item label="确认策略">
                <el-radio-group v-model="form.confirm_policy">
                  <el-radio-button label="on_risk">按需确认</el-radio-button>
                  <el-radio-button label="always">始终询问</el-radio-button>
                  <el-radio-button label="never">全部允许</el-radio-button>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="技能">
                <el-checkbox-group v-model="form.skills">
                  <el-checkbox v-for="sk in meta.skills || []" :key="sk.name" :label="sk.name">
                    {{ sk.name }}
                  </el-checkbox>
                </el-checkbox-group>
              </el-form-item>
              <el-form-item label="上传资料">
                <el-upload drag multiple :auto-upload="false" :on-change="onFileChange" :on-remove="onFileRemove">
                  <div>将文件拖到此处，或点击上传</div>
                </el-upload>
              </el-form-item>
            </el-form>
            <el-button type="primary" :loading="creating" :disabled="!form.goal" @click="submit">
              开始执行
            </el-button>
          </div>

          <div v-else>
            <div style="display: flex; justify-content: space-between; align-items: center; gap: 12px">
              <div>
                <h3 style="margin: 0 0 6px">{{ detail.task.title }}</h3>
                <el-tag :type="statusType(detail.task.status)">{{ statusText(detail.task.status) }}</el-tag>
              </div>
              <div>
                <el-button
                  v-if="detail.task.status === 'running' || detail.task.status === 'waiting_confirm' || detail.task.status === 'queued'"
                  @click="onCancel"
                >
                  取消
                </el-button>
              </div>
            </div>
            <p style="color: #4b5563; white-space: pre-wrap">{{ detail.task.goal }}</p>

            <el-alert
              v-if="detail.task.status === 'waiting_confirm'"
              :title="detail.task.interrupt_info || '需要你确认后才能继续'"
              type="warning"
              show-icon
              :closable="false"
              style="margin-bottom: 12px"
            />
            <div v-if="detail.task.status === 'waiting_confirm'" style="margin-bottom: 16px">
              <el-button type="primary" @click="onConfirm(true)">允许继续</el-button>
              <el-button @click="onConfirm(false)">拒绝</el-button>
            </div>
            <el-alert
              v-if="detail.task.error_message"
              :title="detail.task.error_message"
              type="error"
              show-icon
              :closable="false"
              style="margin-bottom: 12px"
            />

            <h4>执行过程</h4>
            <div v-if="!events.length" class="empty">等待 Agent 事件…</div>
            <div v-for="ev in events" :key="ev.id" class="timeline-item">
              <div class="kicker">{{ formatTime(ev.created_at) }} · {{ ev.agent || 'system' }} · {{ ev.type }}</div>
              <pre>{{ ev.message }}</pre>
            </div>
          </div>
        </section>

        <section class="panel">
          <h4>{{ detail ? '产物' : '能力' }}</h4>
          <div v-if="!detail">
            <p>当前模型：{{ meta.llm?.provider }}/{{ meta.llm?.model }}</p>
            <p class="muted">技能会注入主控 Agent。DeepAgent 负责拆解、读写工作区并调用子代理。</p>
            <div v-for="sk in meta.skills || []" :key="sk.name" class="skill-card">
              <strong>{{ sk.name }}</strong>
              <p>{{ sk.description }}</p>
            </div>
          </div>
          <div v-else>
            <div v-if="!(detail.artifacts || []).length" class="empty">还没有产物。完成后会出现在 output/</div>
            <el-table v-else :data="detail.artifacts" size="small">
              <el-table-column prop="name" label="文件" />
              <el-table-column prop="size" label="大小" width="90" />
              <el-table-column label="">
                <template #default="{ row }">
                  <el-link :href="artifactUrl(detail.task.id, row.id)" target="_blank">下载</el-link>
                </template>
              </el-table-column>
            </el-table>
            <p v-if="detail.task.summary" style="white-space: pre-wrap; margin-top: 16px">
              {{ detail.task.summary }}
            </p>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  artifactUrl,
  cancelTask,
  confirmTask,
  createTask,
  getMeta,
  getTask,
  listTasks,
  openEventStream,
} from '../api'

const meta = ref({ llm: {}, skills: [] })
const tasks = ref([])
const currentId = ref('')
const detail = ref(null)
const events = ref([])
const creating = ref(false)
const form = reactive({
  title: '',
  goal: '',
  confirm_policy: 'on_risk',
  skills: ['weekly-report'],
  files: [],
})

let source = null

const statusText = (s) =>
  ({
    queued: '排队',
    running: '执行中',
    waiting_confirm: '待确认',
    succeeded: '完成',
    failed: '失败',
    cancelled: '已取消',
  })[s] || s

const statusType = (s) =>
  ({
    queued: 'info',
    running: '',
    waiting_confirm: 'warning',
    succeeded: 'success',
    failed: 'danger',
    cancelled: 'info',
  })[s] || 'info'

const formatTime = (v) => {
  if (!v) return ''
  const d = new Date(v)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const resetComposer = () => {
  currentId.value = ''
  detail.value = null
  events.value = []
  closeStream()
}

const onFileChange = (_, fileList) => {
  form.files = fileList.map((f) => f.raw).filter(Boolean)
}

const onFileRemove = (_, fileList) => {
  form.files = fileList.map((f) => f.raw).filter(Boolean)
}

const refreshList = async () => {
  tasks.value = await listTasks()
}

const closeStream = () => {
  if (source) {
    source.close()
    source = null
  }
}

const selectTask = async (id) => {
  currentId.value = id
  const data = await getTask(id)
  detail.value = data
  events.value = data.events || []
  closeStream()
  const last = events.value.at(-1)?.id || 0
  source = openEventStream(id, last, (ev) => {
    if (!events.value.some((x) => x.id === ev.id)) {
      events.value.push(ev)
    }
    if (ev.type === 'status' || ev.type === 'interrupt' || ev.type === 'error') {
      getTask(id).then((fresh) => {
        detail.value = fresh
        refreshList()
      })
    }
  })
}

const submit = async () => {
  creating.value = true
  try {
    const task = await createTask(form)
    ElMessage.success('任务已创建')
    form.goal = ''
    form.title = ''
    form.files = []
    await refreshList()
    await selectTask(task.id)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    creating.value = false
  }
}

const onCancel = async () => {
  await cancelTask(currentId.value)
  await selectTask(currentId.value)
  await refreshList()
}

const onConfirm = async (approved) => {
  await confirmTask(currentId.value, approved)
  await selectTask(currentId.value)
}

onMounted(async () => {
  try {
    meta.value = await getMeta()
    await refreshList()
  } catch (err) {
    ElMessage.error('无法连接后端，请先启动 API 服务')
  }
})

onBeforeUnmount(closeStream)
</script>
