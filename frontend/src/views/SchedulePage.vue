<template>
  <div class="page-shell">
    <header class="topbar">
      <button class="date-switch" @click="shiftDate(-1)"><span class="material-symbols-outlined">chevron_left</span></button>
      <div class="date-center">
        <h1>行程</h1>
        <p>{{ selectedDate }}</p>
      </div>
      <button class="date-switch" @click="shiftDate(1)"><span class="material-symbols-outlined">chevron_right</span></button>
    </header>

    <main class="content">
      <section class="card">
        <h2 class="card-title">当日上班时间</h2>
        <div class="work-hours">09:00 - 18:00</div>
      </section>

      <section class="card">
        <h2 class="card-title">关键行为数据化汇总（次数排序）</h2>
        <div class="summary-list">
          <div v-for="item in sortedSummary" :key="item.key" class="summary-item">
            <span>{{ item.label }}</span>
            <strong>{{ item.count }}</strong>
          </div>
          <div v-if="sortedSummary.length===0" class="empty">暂无行为数据</div>
        </div>
      </section>

      <section class="card">
        <div class="timeline-head">
          <h2 class="card-title" style="margin:0;">时间轴</h2>
          <div class="timeline-actions">
            <van-button size="small" plain type="primary" @click="openCreateModal()">快捷新建</van-button>
            <van-button size="small" type="primary" @click="openCreateModal(selectedDateTimeSlot)">新建当前时段</van-button>
          </div>
        </div>

        <div class="timeline-axis">
          <div
            v-for="slot in timeSlots"
            :key="slot"
            class="timeline-row"
            :class="{ selected: isSlotSelected(slot) }"
            :data-slot="slot"
            @click="openCreateModal(slot)"
          >
            <div class="slot-time">{{ slot }}</div>
            <div class="slot-card">
              <span v-if="eventTitleBySlot(slot)" class="slot-text">{{ eventTitleBySlot(slot) }}</span>
              <span v-else class="slot-placeholder">点击新建事项</span>
            </div>
          </div>
        </div>
      </section>
    </main>

    <van-popup v-model:show="showCreate" position="bottom" round :style="{ height: '58%' }">
      <div class="modal-body">
        <h3>新建事项</h3>
        <van-field v-model="draft.title" label="事项" placeholder="输入事项标题" />
        <van-field v-model="draft.start" label="开始" placeholder="例如 09:00" />
        <van-field v-model="draft.end" label="结束" placeholder="例如 09:30" />
        <van-field v-model="draft.notes" label="备注" placeholder="输入备注或语音转文字" type="textarea" rows="2" autosize />

        <div class="quick-actions">
          <van-button size="small" @click="applyQuick('客户拜访')">客户拜访</van-button>
          <van-button size="small" @click="applyQuick('商务用餐')">商务用餐</van-button>
          <van-button size="small" @click="applyQuick('会议')">会议</van-button>
          <van-button size="small" @click="applyQuick('文档整理')">文档整理</van-button>
        </div>

        <div class="voice-actions">
          <van-button size="small" type="primary" plain @click="speechToText">语音转文字</van-button>
          <van-button size="small" type="primary" @click="voiceStructuredInput">语音结构化录入</van-button>
        </div>

        <van-button type="primary" block style="margin-top: 10px" @click="saveEvent">保存事项</van-button>
      </div>
    </van-popup>

    <AppBottomNav current="schedule" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { showFailToast, showLoadingToast, showSuccessToast, closeToast } from 'vant'
import AppBottomNav from '../components/AppBottomNav.vue'
import { getAttendanceByDate, recognizeVoice, submitStatus } from '../api'
import { STATUS_LABEL, normalizeAttendanceRecord } from '../data/models'

const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const records = ref([])
const selectedDate = ref(new Date().toISOString().slice(0, 10))
const showCreate = ref(false)
const selectionStart = ref('')
const selectionEnd = ref('')

const draft = reactive({ title: '', start: '', end: '', notes: '' })
const scheduleKey = computed(() => `schedule-events-${user.id || 1}-${selectedDate.value}`)
const events = ref([])

const timeSlots = computed(() => {
  const slots = []
  for (let hour = 0; hour <= 23; hour += 1) {
    for (let min = 0; min < 60; min += 15) {
      const slot = `${String(hour).padStart(2, '0')}:${String(min).padStart(2, '0')}`
      slots.push(slot)
    }
  }
  return slots
})

const selectedDateTimeSlot = computed(() => {
  const now = new Date()
  const hour = String(now.getHours()).padStart(2, '0')
  const quarter = Math.floor(now.getMinutes() / 15) * 15
  return `${hour}:${String(quarter).padStart(2, '0')}`
})

const sortedSummary = computed(() => {
  const counter = {}
  records.value.forEach((row) => { counter[row.status] = (counter[row.status] || 0) + 1 })
  return Object.entries(counter).map(([status, count]) => ({ key: status, label: STATUS_LABEL[status] || status, count })).sort((a, b) => b.count - a.count)
})

const loadRecords = async () => {
  const response = await getAttendanceByDate(user.id || 1, selectedDate.value)
  records.value = (response.data.records || []).map(normalizeAttendanceRecord)
}

const loadEvents = () => {
  events.value = JSON.parse(localStorage.getItem(scheduleKey.value) || '[]')
}

const saveEvents = () => {
  localStorage.setItem(scheduleKey.value, JSON.stringify(events.value))
}

const shiftDate = async (dayDiff) => {
  const nextDate = new Date(`${selectedDate.value}T00:00:00`)
  nextDate.setDate(nextDate.getDate() + dayDiff)
  selectedDate.value = nextDate.toISOString().slice(0, 10)
  await loadRecords()
  loadEvents()
}

const openCreateModal = (slot = '') => {
  if (slot) {
    selectionStart.value = slot
    selectionEnd.value = slot
  }
  draft.title = ''
  draft.notes = ''
  draft.start = slot || selectionStart.value || '09:00'
  draft.end = selectionEnd.value || draft.start || '09:15'
  showCreate.value = true
}

const applyQuick = (text) => {
  draft.title = text
  if (!draft.notes) draft.notes = text
}

const speechToText = () => {
  const SpeechRecognitionClass = window.SpeechRecognition || window.webkitSpeechRecognition
  if (!SpeechRecognitionClass) {
    showFailToast('当前浏览器不支持语音转文字')
    return
  }
  const recognition = new SpeechRecognitionClass()
  recognition.lang = 'zh-CN'
  recognition.onresult = (event) => {
    const text = event.results?.[0]?.[0]?.transcript || ''
    draft.notes = text
    if (!draft.title) draft.title = text.slice(0, 10) || '语音事项'
    showSuccessToast('语音文字已填入')
  }
  recognition.onerror = () => showFailToast('语音识别失败')
  recognition.start()
}

const voiceStructuredInput = async () => {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  const mediaRecorder = new MediaRecorder(stream)
  const chunks = []
  mediaRecorder.ondataavailable = (event) => chunks.push(event.data)
  mediaRecorder.onstop = async () => {
    const audioBlob = new Blob(chunks, { type: 'audio/wav' })
    const formData = new FormData()
    formData.append('audio', audioBlob, 'record.wav')
    showLoadingToast({ message: '语音结构化处理中...', forbidClick: true, duration: 0 })
    try {
      const response = await recognizeVoice(formData)
      const result = response.data || {}
      draft.title = STATUS_LABEL[result.status] || result.status || '语音事项'
      draft.notes = `${result.reason || ''} ${result.location || ''}`.trim()
      await submitStatus({
        userId: user.id || 1,
        status: result.status,
        location: result.location,
        reason: result.reason,
        occurredAt: result.occurredAt,
      })
      await loadRecords()
      showSuccessToast('已自动结构化录入系统')
    } catch {
      showFailToast('语音结构化录入失败')
    } finally {
      closeToast()
      stream.getTracks().forEach((track) => track.stop())
    }
  }
  mediaRecorder.start()
  setTimeout(() => mediaRecorder.stop(), 3000)
}

const saveEvent = () => {
  events.value.push({
    id: Date.now(),
    title: draft.title || '未命名事项',
    start: draft.start,
    end: draft.end,
    notes: draft.notes,
  })
  saveEvents()
  showCreate.value = false
  showSuccessToast('事项已创建')
}

const eventTitleBySlot = (slot) => {
  const matchEvent = events.value.find((item) => item.start <= slot && item.end >= slot)
  return matchEvent ? matchEvent.title : ''
}

const isSlotSelected = (slot) => {
  if (!selectionStart.value || !selectionEnd.value) return false
  return slot >= selectionStart.value && slot <= selectionEnd.value
}

onMounted(async () => {
  await loadRecords()
  loadEvents()
})
</script>

<style scoped>
.page-shell { min-height: 100vh; background:#f8f9fa; }
.topbar { height:64px; border-bottom:1px solid #c3c6d7; background:#fff; display:flex; align-items:center; justify-content:space-between; padding:0 12px; }
.date-switch { border:0; background:transparent; width:40px; height:40px; color:#2563eb; border-radius:999px; }
.date-center { text-align:center; }
.date-center h1 { margin:0; font-size:20px; }
.date-center p { margin:2px 0 0; font-size:12px; color:#64748b; }
.content { padding: 14px 14px 84px; display:flex; flex-direction:column; gap:10px; }
.card { border:1px solid #c3c6d7; background:#fff; border-radius:10px; padding:12px; }
.card-title { margin:0 0 8px; font-size:15px; }
.work-hours { color:#004ac6; font-size:18px; font-weight:700; }
.summary-list { display:flex; flex-direction:column; gap:7px; }
.summary-item { display:flex; justify-content:space-between; border:1px solid #e2e8f0; border-radius:8px; padding:8px; font-size:13px; }
.summary-item strong { color:#004ac6; }
.empty { color:#64748b; font-size:13px; }
.timeline-head { display:flex; justify-content:space-between; align-items:center; margin-bottom:8px; }
.timeline-actions { display:flex; gap:8px; }
.timeline-axis { max-height:460px; overflow:auto; display:flex; flex-direction:column; gap:8px; padding-right:2px; }
.timeline-row { display:grid; grid-template-columns:64px minmax(0,1fr); gap:8px; align-items:start; cursor:pointer; }
.timeline-row.selected .slot-card { border-color:#2563eb; background:#eef4ff; }
.slot-time { font-size:12px; color:#64748b; padding-top:10px; position:relative; }
.slot-time::after { content:''; position:absolute; left:56px; top:16px; width:6px; height:6px; border-radius:999px; background:#cbd5e1; }
.slot-card { border:1px solid #e2e8f0; border-radius:10px; min-height:48px; padding:8px 10px; background:#fff; }
.slot-text { display:block; font-size:12px; color:#0f172a; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.slot-placeholder { font-size:12px; color:#94a3b8; }
.modal-body { padding:16px; }
.modal-body h3 { margin:0 0 10px; }
.quick-actions { display:flex; flex-wrap:wrap; gap:6px; margin-top:8px; }
.voice-actions { display:flex; gap:8px; margin-top:10px; }
</style>
