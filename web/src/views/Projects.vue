<template>
  <AppShell>
    <template #side>
      <el-button type="primary" class="side-action" @click="resetComposer">新建项目</el-button>
      <div class="task-list">
        <div
          v-for="item in projects"
          :key="item.id"
          class="task-item"
          :class="{ active: currentId === item.id }"
          @click="select(item.id)"
        >
          <div class="title">{{ item.name }}</div>
          <div class="meta">
            <span>{{ item.description || '无描述' }}</span>
          </div>
        </div>
        <div v-if="!projects.length" class="empty">还没有项目</div>
      </div>
    </template>

    <div class="page-body">
      <section class="panel page-panel">
        <div v-if="!detail">
          <h3>创建项目工作区</h3>
          <p class="muted">项目共享资料会在创建任务时复制到 input/，任务产物也会回写到项目 output/。</p>
          <el-form label-position="top">
            <el-form-item label="名称">
              <el-input v-model="form.name" placeholder="例如：销售周报" />
            </el-form-item>
            <el-form-item label="描述">
              <el-input v-model="form.description" type="textarea" :rows="4" placeholder="这个项目长期做什么" />
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="creating" :disabled="!form.name" @click="submit">创建</el-button>
        </div>
        <div v-else>
          <h3>{{ detail.project.name }}</h3>
          <p class="muted">{{ detail.project.description || '无描述' }}</p>
          <h4>共享资料</h4>
          <el-upload drag multiple :auto-upload="false" :on-change="onFileChange" :on-remove="onFileRemove">
            <div>将文件拖到此处，任务会从这里复制资料</div>
          </el-upload>
          <el-button style="margin: 12px 0" :disabled="!files.length" :loading="uploading" @click="upload">
            上传到项目
          </el-button>
          <el-table v-if="detail.files?.length" :data="detail.files" size="small">
            <el-table-column prop="name" label="文件" />
            <el-table-column prop="size" label="大小" width="100" />
          </el-table>
          <p v-else class="muted">还没有共享文件</p>
          <h4>最近任务</h4>
          <el-table v-if="detail.tasks?.length" :data="detail.tasks" size="small">
            <el-table-column prop="title" label="标题" />
            <el-table-column prop="status" label="状态" width="120" />
          </el-table>
          <p v-else class="muted">还没有关联任务</p>
        </div>
      </section>
    </div>
  </AppShell>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/AppShell.vue'
import { createProject, getMeta, getProject, listProjects, uploadProjectFiles } from '../api'

const meta = ref({ llm: {} })
const projects = ref([])
const currentId = ref('')
const detail = ref(null)
const creating = ref(false)
const uploading = ref(false)
const files = ref([])
const form = reactive({ name: '', description: '' })

const resetComposer = () => {
  currentId.value = ''
  detail.value = null
}

const refresh = async () => {
  projects.value = await listProjects()
}

const select = async (id) => {
  currentId.value = id
  detail.value = await getProject(id)
}

const submit = async () => {
  creating.value = true
  try {
    const item = await createProject(form)
    form.name = ''
    form.description = ''
    ElMessage.success('项目已创建')
    await refresh()
    await select(item.id)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    creating.value = false
  }
}

const onFileChange = (_, fileList) => {
  files.value = fileList.map((f) => f.raw).filter(Boolean)
}

const onFileRemove = (_, fileList) => {
  files.value = fileList.map((f) => f.raw).filter(Boolean)
}

const upload = async () => {
  uploading.value = true
  try {
    await uploadProjectFiles(currentId.value, files.value)
    files.value = []
    ElMessage.success('已上传到项目')
    await select(currentId.value)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    uploading.value = false
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
