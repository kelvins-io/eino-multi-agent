<template>
  <AppShell>
    <template #side>
      <el-button type="primary" class="side-action" @click="resetComposer">新建定时任务</el-button>
      <div class="task-list">
        <div
          v-for="item in schedules"
          :key="item.id"
          class="task-item"
          :class="{ active: currentId === item.id }"
          @click="currentId = item.id"
        >
          <div class="title">{{ item.title }}</div>
          <div class="meta">
            <el-tag size="small" :type="item.enabled ? 'success' : 'info'">{{ item.enabled ? '启用' : '停用' }}</el-tag>
            <span>{{ kindText(item) }}</span>
          </div>
        </div>
        <div v-if="!schedules.length" class="empty">还没有定时任务</div>
      </div>
    </template>

    <div class="page-body">
      <section class="panel page-panel">
        <div v-if="!current">
          <h3>定时创建工作任务</h3>
          <p class="muted">支持一次性、每天、每周。演示时可点「立即运行」马上生成一条任务。</p>
          <el-form label-position="top">
            <el-form-item label="标题">
              <el-input v-model="form.title" placeholder="可空，默认截取目标" />
            </el-form-item>
            <el-form-item label="目标">
              <el-input v-model="form.goal" type="textarea" :rows="5" placeholder="到点要完成什么" />
            </el-form-item>
            <el-form-item label="所属项目">
              <el-select v-model="form.project_id" clearable placeholder="可选" style="width: 100%">
                <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="周期">
              <el-radio-group v-model="form.kind">
                <el-radio-button label="once">一次性</el-radio-button>
                <el-radio-button label="daily">每天</el-radio-button>
                <el-radio-button label="weekly">每周</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.kind === 'weekly'" label="星期">
              <el-select v-model="form.weekday" style="width: 100%">
                <el-option v-for="d in weekdays" :key="d.value" :label="d.label" :value="d.value" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="form.kind !== 'once'" label="时间">
              <el-input-number v-model="form.hour" :min="0" :max="23" />
              <span class="muted" style="margin: 0 8px">:</span>
              <el-input-number v-model="form.minute" :min="0" :max="59" />
            </el-form-item>
            <el-form-item v-else label="运行时间">
              <el-date-picker v-model="form.run_at" type="datetime" placeholder="选择时间" style="width: 100%" />
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
                <el-checkbox v-for="sk in meta.skills || []" :key="sk.name" :label="sk.name">{{ sk.name }}</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="creating" :disabled="!form.goal || (form.kind === 'once' && !form.run_at)" @click="submit">保存</el-button>
        </div>
        <div v-else>
          <div style="display: flex; justify-content: space-between; align-items: center; gap: 12px">
            <div>
              <h3 style="margin: 0 0 6px">{{ current.title }}</h3>
              <el-tag :type="current.enabled ? 'success' : 'info'">{{ current.enabled ? '启用' : '停用' }}</el-tag>
              <el-tag size="small" style="margin-left: 6px">{{ kindText(current) }}</el-tag>
            </div>
            <div>
              <el-button @click="onToggle">{{ current.enabled ? '停用' : '启用' }}</el-button>
              <el-button type="primary" :loading="running" @click="onRun">立即运行</el-button>
            </div>
          </div>
          <p style="white-space: pre-wrap">{{ current.goal }}</p>
          <p class="muted">下次运行：{{ formatTime(current.next_run_at) || '无' }}</p>
          <p class="muted">上次运行：{{ formatTime(current.last_run_at) || '尚未运行' }}</p>
        </div>
      </section>
    </div>
  </AppShell>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/AppShell.vue'
import { createSchedule, getMeta, listProjects, listSchedules, runSchedule, toggleSchedule } from '../api'

const weekdays = [
  { value: 0, label: '周日' },
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
]

const meta = ref({ llm: {}, skills: [] })
const projects = ref([])
const schedules = ref([])
const currentId = ref('')
const creating = ref(false)
const running = ref(false)
const form = reactive({
  title: '',
  goal: '',
  project_id: '',
  kind: 'weekly',
  weekday: 5,
  hour: 18,
  minute: 0,
  run_at: '',
  confirm_policy: 'on_risk',
  skills: ['weekly-report'],
})

const current = computed(() => schedules.value.find((s) => s.id === currentId.value) || null)

const kindText = (item) => {
  if (!item) return ''
  if (item.kind === 'once') return '一次性'
  if (item.kind === 'daily') return `每天 ${pad(item.hour)}:${pad(item.minute)}`
  const day = weekdays.find((d) => d.value === item.weekday)?.label || `周${item.weekday}`
  return `${day} ${pad(item.hour)}:${pad(item.minute)}`
}

const pad = (n) => String(n).padStart(2, '0')

const formatTime = (v) => {
  if (!v) return ''
  const d = new Date(v)
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const resetComposer = () => {
  currentId.value = ''
}

const refresh = async () => {
  schedules.value = await listSchedules()
}

const submit = async () => {
  creating.value = true
  try {
    const payload = { ...form }
    if (payload.kind === 'once' && payload.run_at) {
      payload.run_at = new Date(payload.run_at).toISOString()
    } else {
      payload.run_at = ''
    }
    const item = await createSchedule(payload)
    ElMessage.success('定时任务已保存')
    form.goal = ''
    form.title = ''
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
    const item = await toggleSchedule(currentId.value)
    ElMessage.success(item.enabled ? '已启用' : '已停用')
    await refresh()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

const onRun = async () => {
  running.value = true
  try {
    const task = await runSchedule(currentId.value)
    ElMessage.success(`已创建任务 ${task.title || task.id}`)
    await refresh()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    running.value = false
  }
}

onMounted(async () => {
  try {
    meta.value = await getMeta()
    projects.value = await listProjects()
    await refresh()
  } catch (err) {
    ElMessage.error('无法连接后端，请先启动 API 服务')
  }
})
</script>
