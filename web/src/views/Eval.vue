<template>
  <AppShell>
    <template #side>
      <p class="muted" style="padding: 0 4px">按历史任务评估周报 / 调研 / 文件整理的完成率和确认次数，不重新跑模型。</p>
    </template>
    <div class="page-body">
      <section class="panel page-panel">
        <div class="eval-head">
          <h3 style="margin: 0">评估套件</h3>
          <el-button :loading="loading" @click="refresh">刷新</el-button>
        </div>
        <p class="muted">总完成率 {{ pct(report.completion) }} · {{ report.passed }}/{{ report.total }} · 人工确认 {{ report.confirms }} 次</p>
        <div v-for="block in report.cases || []" :key="block.case.id" class="eval-case">
          <div class="todo-head">
            <strong>{{ block.case.name }}</strong>
            <span>{{ pct(block.completion) }} · {{ block.passed }}/{{ block.total }}</span>
          </div>
          <p class="muted">技能 {{ block.case.skill }} · 预期产物 {{ (block.case.expect_files || []).join('、') || '任意 output 文件' }}</p>
          <el-table v-if="block.items?.length" :data="block.items" size="small">
            <el-table-column prop="title" label="任务" />
            <el-table-column prop="status" label="状态" width="110" />
            <el-table-column prop="confirms" label="确认" width="70" />
            <el-table-column label="结果" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.passed ? 'success' : 'info'">{{ row.passed ? '通过' : '未过' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="说明" />
          </el-table>
          <p v-else class="muted">还没有匹配该技能的历史任务</p>
        </div>
      </section>
    </div>
  </AppShell>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/AppShell.vue'
import { getEval, getMeta } from '../api'

const meta = ref({ llm: {} })
const loading = ref(false)
const report = ref({ cases: [], total: 0, passed: 0, completion: 0, confirms: 0 })

const pct = (n) => `${Math.round((n || 0) * 100)}%`

const refresh = async () => {
  loading.value = true
  try {
    report.value = await getEval()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
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
