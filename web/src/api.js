import axios from 'axios'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

export const getMeta = () => http.get('/meta').then((r) => r.data)
export const listTasks = () => http.get('/tasks').then((r) => r.data.items || [])
export const getTask = (id) => http.get(`/tasks/${id}`).then((r) => r.data)

export const createTask = (payload) => {
  const form = new FormData()
  form.append('title', payload.title || '')
  form.append('goal', payload.goal)
  form.append('confirm_policy', payload.confirm_policy)
  form.append('skills', JSON.stringify(payload.skills || []))
  if (payload.project_id) {
    form.append('project_id', payload.project_id)
  }
  for (const file of payload.files || []) {
    form.append('files', file)
  }
  return http.post('/tasks', form).then((r) => r.data)
}

export const listProjects = () => http.get('/projects').then((r) => r.data.items || [])
export const getProject = (id) => http.get(`/projects/${id}`).then((r) => r.data)
export const createProject = (payload) => http.post('/projects', payload).then((r) => r.data)
export const uploadProjectFiles = (id, files) => {
  const form = new FormData()
  for (const file of files || []) {
    form.append('files', file)
  }
  return http.post(`/projects/${id}/files`, form).then((r) => r.data)
}

export const listSchedules = () => http.get('/schedules').then((r) => r.data.items || [])
export const createSchedule = (payload) => http.post('/schedules', payload).then((r) => r.data)
export const runSchedule = (id) => http.post(`/schedules/${id}/run`).then((r) => r.data)
export const toggleSchedule = (id) => http.post(`/schedules/${id}/toggle`).then((r) => r.data)

export const listConnectors = () => http.get('/connectors').then((r) => r.data.items || [])
export const createConnector = (payload) => http.post('/connectors', payload).then((r) => r.data)
export const toggleConnector = (id) => http.post(`/connectors/${id}/toggle`).then((r) => r.data)
export const testConnector = (id) => http.post(`/connectors/${id}/test`).then((r) => r.data)

export const listAudit = () => http.get('/audit').then((r) => r.data.items || [])
export const getEval = () => http.get('/eval').then((r) => r.data)

export const cancelTask = (id) => http.post(`/tasks/${id}/cancel`).then((r) => r.data)
export const retryTask = (id) => http.post(`/tasks/${id}/retry`).then((r) => r.data)
export const confirmTask = (id, approved) =>
  http.post(`/tasks/${id}/confirm`, { approved }).then((r) => r.data)

export const artifactUrl = (taskId, artifactId, inline = false) =>
  `/api/v1/tasks/${taskId}/artifacts/${artifactId}${inline ? '?inline=1' : ''}`
export const getArtifactPreview = (taskId, artifactId) =>
  http.get(`/tasks/${taskId}/artifacts/${artifactId}/preview`).then((r) => r.data)

export const openEventStream = (taskId, after, onEvent) => {
  const url = `/api/v1/tasks/${taskId}/events?after=${after || 0}`
  const es = new EventSource(url)
  es.addEventListener('task', (e) => {
    try {
      onEvent(JSON.parse(e.data))
    } catch {
      /* ignore */
    }
  })
  return es
}
