import axios from 'axios'

const envBase = import.meta.env.VITE_API_BASE_URL
const apiBaseURL = typeof envBase === 'string' && envBase.length > 0 ? envBase : ''

const api = axios.create({
  baseURL: apiBaseURL,
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
export const getAuthUsers = () => api.get('/v1/auth/users')
export const changePassword = (payload) => api.post('/v1/auth/password/change', payload)
export const resetUserPassword = (payload) => api.post('/v1/auth/password/reset', payload)
export const recognizeVoice = (formData) => api.post('/v1/voice/recognize', formData)
export const submitStatus = (payload) => api.post('/v1/attendance/submit', payload)
export const getTodayRecords = (userId) => api.get('/v1/attendance/today', { params: { userId } })
export const getAttendanceSummary = (userId, period) => api.get('/v1/attendance/summary', { params: { userId, period } })
export const getAttendanceList = (userId, period) => api.get('/v1/attendance/list', { params: { userId, period } })
export const getAttendanceByDate = (userId, date) => api.get('/v1/attendance/by-date', { params: { userId, date } })
export const getLocationCatalog = () => api.get('/v1/catalog/locations')
export const getScheduleQuick = () => api.get('/v1/catalog/schedule-quick')

export const getScheduleDayOptions = () => api.get('/v1/schedule/day-options')
export const getScheduleDay = (userId, date) => api.get('/v1/schedule/day', { params: { userId, date } })
export const setScheduleDay = (payload) => api.post('/v1/schedule/day', payload)

export const getAdminLocationCatalog = (params) => {
  const useParams = params && typeof params === 'object' && Object.keys(params).length > 0
  return api.get('/v1/admin/data/location-catalog', useParams ? { params } : {})
}
export const upsertAdminLocationCatalog = (payload) =>
  api.post('/v1/admin/data/location-catalog/upsert', payload)
export const deleteAdminLocationCatalog = (payload) =>
  api.post('/v1/admin/data/location-catalog/delete', payload)
export const getAdminScheduleQuickSections = () => api.get('/v1/admin/data/schedule-quick-sections')
export const upsertAdminScheduleQuickSection = (payload) =>
  api.post('/v1/admin/data/schedule-quick-sections/upsert', payload)
export const deleteAdminScheduleQuickSection = (payload) =>
  api.post('/v1/admin/data/schedule-quick-sections/delete', payload)
export const getAdminBoard = () => api.get('/v1/admin/board')
export const getAdminSummary = (period) => api.get('/v1/admin/summary', { params: { period } })
export const getAdminReport = () => api.get('/v1/admin/report')
export const getAdminTeam = (period, date) => api.get('/v1/admin/team', { params: { period, date } })

export const getAdminShiftTypes = () => api.get('/v1/admin/data/shift-types')
export const upsertAdminShiftType = (payload) => api.post('/v1/admin/data/shift-types/upsert', payload)
export const getAdminActivityTypes = () => api.get('/v1/admin/data/activity-types')
export const upsertAdminActivityType = (payload) => api.post('/v1/admin/data/activity-types/upsert', payload)
