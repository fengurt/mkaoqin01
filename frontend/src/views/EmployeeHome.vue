<template>
  <div class="st-emp-page">
    <header class="st-topbar">
      <div class="st-topbar-left">
        <div class="st-avatar-wrap">
          <img alt="User" class="st-avatar" src="https://fastly.jsdelivr.net/npm/@vant/assets/cat.jpeg" />
        </div>
        <h1 class="st-brand">Intervoice</h1>
      </div>
      <button class="st-icon-btn" type="button">
        <span class="material-symbols-outlined">notifications</span>
      </button>
    </header>

    <main class="st-main">
      <div class="st-title-wrap">
        <span class="material-symbols-outlined st-title-icon">dashboard</span>
        <h2 class="st-section-title">状态门户</h2>
      </div>

      <div class="st-action-grid">
        <button
          v-for="statusItem in statusList"
          :key="statusItem"
          class="st-action-card"
          type="button"
          @click="setStatus(statusItem)"
        >
          <div class="st-action-icon-wrap" :class="{ 'st-action-icon-active': form.status === statusItem }">
            <span class="material-symbols-outlined st-action-icon">{{ statusIconMap[statusItem] }}</span>
          </div>
          <span class="st-action-label">{{ statusMap[statusItem] }}</span>
        </button>
      </div>

      <div class="st-title-wrap">
        <span class="material-symbols-outlined st-title-icon">timeline</span>
        <h3 class="st-sub-title">今日动态</h3>
      </div>
      <div class="st-timeline-card">
        <div class="st-timeline-head">今天</div>
        <div v-if="records.length === 0" class="st-empty">暂无记录</div>
        <div v-else class="st-timeline-list">
          <div v-for="record in records" :key="record.id" class="st-timeline-item">
            <div class="st-timeline-dot">
              <span class="material-symbols-outlined">{{ statusIconMap[record.status] || 'event' }}</span>
            </div>
            <div class="st-timeline-content">
              <div class="st-timeline-row">
                <strong>{{ statusMap[record.status] || record.status }}</strong>
                <span>{{ formatTime(record.occurredAt) }}</span>
              </div>
              <p>{{ record.location }} · {{ record.reason }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="card" style="margin-top: 12px; margin-bottom: 90px">
        <div class="section-title">AI 识别与确认</div>
        <van-field v-model="form.status" label="状态编码" placeholder="AI 自动填充" />
        <van-field v-model="form.location" label="地点" placeholder="AI 自动填充" />
        <van-field v-model="form.reason" label="事由" placeholder="AI 自动填充" />
        <van-button type="primary" block round style="margin-top: 8px" @click="handleSubmit">确认并提交</van-button>
      </div>

      <div class="card" style="margin-bottom: 90px">
        <div class="section-title">考勤视图导航</div>
        <van-space wrap>
          <van-button size="small" @click="$router.push('/my-attendance/day')">我的考勤-日</van-button>
          <van-button size="small" @click="$router.push('/my-attendance/week')">我的考勤-周</van-button>
          <van-button size="small" @click="$router.push('/my-attendance/month')">我的考勤-月</van-button>
          <van-button size="small" @click="$router.push('/employee/profile')">员工资料</van-button>
        </van-space>
      </div>
    </main>

    <nav class="st-bottom-nav">
      <a class="st-nav-item st-nav-active" href="#"><span class="material-symbols-outlined">dashboard</span><span>首页</span></a>
      <a class="st-nav-item" href="#"><span class="material-symbols-outlined">mic</span><span>语音</span></a>
      <a class="st-nav-item" href="#"><span class="material-symbols-outlined">analytics</span><span>团队</span></a>
      <a class="st-nav-item" href="#"><span class="material-symbols-outlined">person</span><span>我的</span></a>
    </nav>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { showLoadingToast, showSuccessToast, closeToast } from 'vant'
import { getTodayRecords, recognizeVoice, submitStatus } from '../api'

const statusList = ['CHECK_IN', 'OFFICE', 'OUTING', 'DINING', 'BUSINESS_TRIP', 'CHECK_OUT']
const statusMap = {
  CHECK_IN: '签到',
  OFFICE: '在办公室',
  OUTING: '外出拜访',
  DINING: '商务用餐',
  BUSINESS_TRIP: '出差',
  CHECK_OUT: '签退',
}
const statusIconMap = {
  CHECK_IN: 'login',
  OFFICE: 'apartment',
  OUTING: 'directions_run',
  DINING: 'restaurant',
  BUSINESS_TRIP: 'flight_takeoff',
  CHECK_OUT: 'logout',
}

const records = ref([])
const storedUser = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const form = reactive({ status: 'OFFICE', location: '', reason: '', occurredAt: '' })

const setStatus = (statusValue) => { form.status = statusValue }

const loadRecords = async () => {
  const response = await getTodayRecords(storedUser.id)
  records.value = response.data.records || []
}

const handleVoiceRecorded = async (audioBlob) => {
  showLoadingToast({ message: '正在识别中...', forbidClick: true, duration: 0 })
  const formData = new FormData()
  formData.append('audio', audioBlob, 'record.wav')
  const response = await recognizeVoice(formData)
  closeToast()
  form.status = response.data.status
  form.location = response.data.location
  form.reason = response.data.reason
  form.occurredAt = response.data.occurredAt
  showSuccessToast('已完成 AI 自动填充')
}

const handleSubmit = async () => {
  await submitStatus({
    userId: storedUser.id,
    status: form.status,
    location: form.location,
    reason: form.reason,
    occurredAt: form.occurredAt,
  })
  showSuccessToast('提交成功')
  await loadRecords()
}

const formatTime = (timeValue) => (timeValue ? new Date(timeValue).toLocaleTimeString() : '-')

onMounted(async () => {
  await loadRecords()
  // Hidden press-to-talk per requirement, keep method for future switch.
  void handleVoiceRecorded
})
</script>

<style scoped>
.st-emp-page { background: #f8f9fa; min-height: 100vh; color: #191c1d; }
.st-topbar {
  height: 64px; padding: 0 16px; border-bottom: 1px solid #c3c6d7; background: #fff;
  display: flex; justify-content: space-between; align-items: center; position: sticky; top: 0; z-index: 20;
}
.st-topbar-left { display: flex; align-items: center; gap: 10px; }
.st-avatar-wrap { width: 40px; height: 40px; border-radius: 9999px; overflow: hidden; border: 1px solid #c3c6d7; }
.st-avatar { width: 100%; height: 100%; object-fit: cover; }
.st-brand { margin: 0; color: #004ac6; font-size: 30px; line-height: 38px; }
.st-icon-btn { width: 40px; height: 40px; border-radius: 999px; border: none; background: transparent; color: #434655; }
.st-main { padding: 24px 16px; max-width: 1280px; margin: 0 auto; }
.st-title-wrap { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.st-title-icon { color: #004ac6; font-size: 22px; }
.st-section-title { margin: 0 0 16px; font-size: 24px; }
.st-action-grid { display: grid; grid-template-columns: repeat(3, minmax(0,1fr)); gap: 8px; margin-bottom: 24px; }
.st-action-card {
  border: 1px solid #c3c6d7; background: #fff; border-radius: 8px; padding: 12px 8px;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
}
.st-action-icon-wrap { width: 48px; height: 48px; border-radius: 999px; background: rgba(37,99,235,0.1); display: flex; align-items: center; justify-content: center; }
.st-action-icon-active { background: rgba(37,99,235,0.22); }
.st-action-icon { font-size: 24px; color: #2563eb; }
.st-action-label { font-size: 12px; font-weight: 600; text-align: center; }
.st-sub-title { margin: 0 0 8px; font-size: 20px; }
.st-timeline-card { border: 1px solid #c3c6d7; border-radius: 8px; background: #fff; overflow: hidden; }
.st-timeline-head { padding: 10px 12px; border-bottom: 1px solid #c3c6d7; background: #f8f9fa; font-size: 12px; font-weight: 600; color: #434655; }
.st-empty { padding: 16px; color: #64748b; font-size: 14px; }
.st-timeline-list { padding: 12px; display: flex; flex-direction: column; gap: 12px; }
.st-timeline-item { display: flex; gap: 12px; }
.st-timeline-dot { width: 32px; height: 32px; border-radius: 999px; background: #2563eb; color: #fff; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.st-timeline-dot .material-symbols-outlined { font-size: 16px; }
.st-timeline-content { flex: 1; }
.st-timeline-row { display: flex; justify-content: space-between; align-items: center; font-size: 12px; }
.st-timeline-content p { margin: 4px 0 0; font-size: 13px; color: #434655; }
.st-bottom-nav {
  position: fixed; left: 0; right: 0; bottom: 0; height: 64px; background: #fff; border-top: 1px solid #c3c6d7;
  display: flex; align-items: center; justify-content: space-around; z-index: 50;
}
.st-nav-item { display: flex; flex-direction: column; align-items: center; gap: 2px; color: #434655; text-decoration: none; font-size: 12px; }
.st-nav-active { color: #004ac6; font-weight: 700; }
</style>
