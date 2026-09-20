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
  for (const file of payload.files || []) {
    form.append('files', file)
  }
  return http.post('/tasks', form).then((r) => r.data)
}

export const cancelTask = (id) => http.post(`/tasks/${id}/cancel`).then((r) => r.data)
export const confirmTask = (id, approved) =>
  http.post(`/tasks/${id}/confirm`, { approved }).then((r) => r.data)

export const artifactUrl = (taskId, artifactId) => `/api/v1/tasks/${taskId}/artifacts/${artifactId}`

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
