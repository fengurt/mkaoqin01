import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:8010',
  timeout: 20000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export const login = (payload) => api.post('/v1/auth/login', payload)
export const wechatLogin = () => api.post('/v1/auth/wechat', { code: 'demo' })
export const recognizeVoice = (formData) => api.post('/v1/voice/recognize', formData)
export const submitStatus = (payload) => api.post('/v1/attendance/submit', payload)
export const getTodayRecords = (userId) => api.get('/v1/attendance/today', { params: { userId } })
export const getAttendanceSummary = (userId, period) => api.get('/v1/attendance/summary', { params: { userId, period } })
export const getAttendanceList = (userId, period) => api.get('/v1/attendance/list', { params: { userId, period } })
export const getAttendanceByDate = (userId, date) => api.get('/v1/attendance/by-date', { params: { userId, date } })
export const getAdminBoard = () => api.get('/v1/admin/board')
export const getAdminSummary = (period) => api.get('/v1/admin/summary', { params: { period } })
export const getAdminReport = () => api.get('/v1/admin/report')
export const getAdminTeam = (period, date) => api.get('/v1/admin/team', { params: { period, date } })
