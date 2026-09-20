<template>
  <AppShell :model="meta.llm?.model">
    <template #side>
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
    </template>
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
              <el-form-item label="所属项目">
                <el-select v-model="form.project_id" clearable placeholder="可选，共享资料会复制到本次任务" style="width: 100%">
                  <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
                </el-select>
              </el-form-item>
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
                <el-tag size="small" style="margin-left: 6px">{{ policyText(detail.task.confirm_policy) }}</el-tag>
              </div>
              <div>
                <el-button
                  v-if="detail.task.status === 'running' || detail.task.status === 'waiting_confirm' || detail.task.status === 'queued'"
                  @click="onCancel"
                >
                  取消
                </el-button>
                <el-button
                  v-if="detail.task.status === 'failed' || detail.task.status === 'cancelled' || detail.task.status === 'succeeded'"
                  type="primary"
                  :loading="retrying"
                  @click="onRetry"
                >
                  重新执行
                </el-button>
              </div>
            </div>
            <p style="color: #4b5563; white-space: pre-wrap">{{ detail.task.goal }}</p>

            <div v-if="detail.task.status === 'waiting_confirm'" class="confirm-card">
              <div class="confirm-title">需要你确认后才能继续</div>
              <p>{{ detail.task.interrupt_info || '该操作被确认策略拦截。' }}</p>
              <p class="muted">允许后继续执行；拒绝会停止本次任务，可稍后重新执行。</p>
              <div class="confirm-actions">
                <el-button type="primary" @click="onConfirm(true)">允许继续</el-button>
                <el-button type="danger" plain @click="onConfirm(false)">拒绝并停止</el-button>
              </div>
            </div>
            <el-alert
              v-if="detail.task.error_message"
              :title="detail.task.error_message"
              type="error"
              show-icon
              :closable="false"
              style="margin-bottom: 12px"
            />

            <div v-if="todos.length" class="todo-progress">
              <div class="todo-head">
                <span>任务拆解</span>
                <span>{{ todoDone }}/{{ todos.length }}</span>
              </div>
              <el-progress :percentage="todoPct" :stroke-width="8" />
              <ul class="todo-list">
                <li v-for="(item, idx) in todos" :key="idx" :class="item.status">
                  {{ item.content }}
                </li>
              </ul>
            </div>

            <h4>执行过程</h4>
            <div v-if="!events.length" class="empty">等待 Agent 事件…</div>
            <div v-for="ev in events" :key="ev.id" class="timeline-item" :class="'kind-' + ev.type">
              <div class="kicker">{{ formatTime(ev.created_at) }} · {{ ev.agent || 'system' }} · {{ stepLabel(ev) }}</div>
              <pre v-if="ev.message">{{ ev.message }}</pre>
            </div>
          </div>
        </section>

        <section class="panel">
          <h4>{{ detail ? '产物' : '能力' }}</h4>
          <div v-if="!detail">
            <p>当前模型：{{ meta.llm?.provider }}/{{ meta.llm?.model }}</p>
            <p class="muted">技能用 skill 工具按需加载。主控拆解任务，research / office / code 子代理分别负责调研、文档表格和数据分析。覆盖已有文件会按确认策略暂停。</p>
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
              <el-table-column label="" width="140">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openPreview(row)">预览</el-button>
                  <el-link :href="artifactUrl(detail.task.id, row.id)" target="_blank">下载</el-link>
                </template>
              </el-table-column>
            </el-table>
            <p v-if="detail.task.summary" style="white-space: pre-wrap; margin-top: 16px">
              {{ detail.task.summary }}
            </p>
            <el-drawer v-model="preview.visible" :title="preview.name" size="55%">
              <p v-if="preview.loading" class="muted">加载预览…</p>
              <img v-else-if="preview.kind === 'image'" :src="preview.url" alt="" class="preview-image" />
              <pre v-else-if="preview.text" class="preview-text">{{ preview.text }}</pre>
              <p v-else class="muted">该文件不支持预览，请下载查看。</p>
            </el-drawer>
          </div>
        </section>
      </div>
  </AppShell>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/AppShell.vue'
import {
  artifactUrl,
  cancelTask,
  confirmTask,
  createTask,
  getArtifactPreview,
  getMeta,
  getTask,
  listProjects,
  listTasks,
  openEventStream,
  retryTask,
} from '../api'

const meta = ref({ llm: {}, skills: [] })
const tasks = ref([])
const currentId = ref('')
const detail = ref(null)
const events = ref([])
const creating = ref(false)
const retrying = ref(false)
const preview = reactive({ visible: false, loading: false, name: '', kind: '', text: '', url: '' })
const projects = ref([])
const form = reactive({
  title: '',
  goal: '',
  confirm_policy: 'on_risk',
  skills: ['weekly-report'],
  project_id: '',
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

const policyText = (s) =>
  ({
    on_risk: '按需确认',
    always: '始终询问',
    never: '全部允许',
  })[s] || s

const todos = computed(() => detail.value?.task?.todos || [])
const todoDone = computed(() => todos.value.filter((t) => t.status === 'completed').length)
const todoPct = computed(() => (todos.value.length ? Math.round((todoDone.value / todos.value.length) * 100) : 0))

const stepLabel = (ev) =>
  ({
    think: '思考',
    read: '读文件',
    write: '写文件',
    search: '搜索',
    shell: '命令',
    skill: '技能',
    todos: '拆解',
    interrupt: '等待确认',
    confirm: '确认',
    status: '状态',
    error: '错误',
    system: '系统',
    tool_call: '调用工具',
    tool_result: '工具结果',
    assistant: '回复',
  })[ev.type] || ev.type

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
    if (['status', 'interrupt', 'error', 'todos', 'write'].includes(ev.type)) {
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
  try {
    await confirmTask(currentId.value, approved)
    ElMessage.success(approved ? '已允许，继续执行' : '已拒绝，任务已停止')
    await selectTask(currentId.value)
    await refreshList()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

const onRetry = async () => {
  retrying.value = true
  try {
    await retryTask(currentId.value)
    ElMessage.success('已重新执行')
    await selectTask(currentId.value)
    await refreshList()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    retrying.value = false
  }
}

const openPreview = async (row) => {
  preview.visible = true
  preview.loading = true
  preview.name = row.name
  preview.kind = ''
  preview.text = ''
  preview.url = ''
  try {
    const data = await getArtifactPreview(currentId.value, row.id)
    if (data.kind === 'image') {
      preview.kind = 'image'
      preview.url = data.url
    } else if (data.text) {
      preview.kind = 'text'
      preview.text = data.truncated ? `${data.text}\n…[已截断]` : data.text
    } else {
      preview.kind = 'none'
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
    preview.visible = false
  } finally {
    preview.loading = false
  }
}

onMounted(async () => {
  try {
    meta.value = await getMeta()
    projects.value = await listProjects()
    await refreshList()
  } catch (err) {
    ElMessage.error('无法连接后端，请先启动 API 服务')
  }
})

onBeforeUnmount(closeStream)
</script>
