<template>
  <AppShell>
    <template #side>
      <p class="muted" style="padding: 0 4px">记录创建、确认、续跑等操作，进程重启也会写入。</p>
    </template>
    <div class="page-body">
      <section class="panel page-panel">
        <h3>审计日志</h3>
        <el-table v-if="items.length" :data="items" size="small">
          <el-table-column prop="created_at" label="时间" width="160">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="actor" label="操作者" width="100" />
          <el-table-column prop="action" label="动作" width="160" />
          <el-table-column prop="target_type" label="对象" width="100" />
          <el-table-column prop="detail" label="说明" />
        </el-table>
        <p v-else class="muted">还没有审计记录</p>
      </section>
    </div>
  </AppShell>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/AppShell.vue'
import { getMeta, listAudit } from '../api'

const meta = ref({ llm: {} })
const items = ref([])

const formatTime = (v) => {
  if (!v) return ''
  const d = new Date(v)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

onMounted(async () => {
  try {
    meta.value = await getMeta()
    items.value = await listAudit()
  } catch (err) {
    ElMessage.error('无法连接后端，请先启动 API 服务')
  }
})
</script>
