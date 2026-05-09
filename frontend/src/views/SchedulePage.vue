<template>
  <div class="page-shell">
    <header class="topbar schedule-topbar">
      <h1 class="topbar-title">行程</h1>
    </header>

    <main class="content">
      <section class="card calendar-card">
        <div class="calendar-toolbar">
          <button type="button" class="week-nav-btn" aria-label="上一周" @click="shiftWeek(-1)">
            <span class="material-symbols-outlined" aria-hidden="true">chevron_left</span>
          </button>
          <button type="button" class="ymd-button" @click="showMonthCalendar = true">
            {{ formattedYmd }}
          </button>
          <button type="button" class="week-nav-btn" aria-label="下一周" @click="shiftWeek(1)">
            <span class="material-symbols-outlined" aria-hidden="true">chevron_right</span>
          </button>
        </div>
        <div class="week-strip" role="list" aria-label="本周日期">
          <button
            v-for="day in weekDays"
            :key="day.date"
            type="button"
            role="listitem"
            class="week-day-cell"
            :class="{ selected: day.isSelected, today: day.isToday }"
            :aria-pressed="day.isSelected"
            :aria-current="day.isSelected ? 'true' : undefined"
            @click="selectScheduleDay(day.date)"
          >
            <span class="week-day-name">{{ day.weekdayShort }}</span>
            <strong class="week-day-num">{{ day.dayNum }}</strong>
          </button>
        </div>
      </section>

      <section class="card timeline-card">
        <div class="timeline-head timeline-head-compact">
          <div class="timeline-head-left">
            <h2 class="card-title timeline-title">时间轴</h2>
            <button
              type="button"
              class="work-hours-pill"
              :title="daySchedule?.mode === 'leave' ? '当日休假类型，点击更改' : '当日常规班班次（与后台班次一致），点击更改'"
              @click="showSchedulePicker = true"
            >
              {{ schedulePillLabel }}
            </button>
          </div>
          <div class="timeline-head-right">
            <div class="timeline-actions">
              <van-button size="small" plain type="primary" @click="openCreateModal()">快捷新建</van-button>
              <van-button size="small" type="primary" @click="openCreateModal(selectedDateTimeSlot)">新建当前时段</van-button>
            </div>
          </div>
        </div>

        <div v-if="sortedSummary.length" class="summary-row" aria-label="关键行为汇总">
          <span class="summary-row-label">汇总</span>
          <div class="summary-chips">
            <span v-for="item in sortedSummary" :key="item.key" class="summary-chip">{{ item.label }} ×{{ item.count }}</span>
          </div>
        </div>
        <div v-else class="summary-empty">暂无行为数据</div>

        <div class="timeline-body">
          <div class="period-tabs" role="tablist" aria-label="时段快捷跳转">
            <button
              v-for="preset in periodPresetsLive"
              :key="preset.key"
              type="button"
              class="period-tab"
              :class="{ active: activePeriod === preset.key }"
              role="tab"
              :aria-selected="activePeriod === preset.key"
              @click="jumpToPeriod(preset.key)"
            >
              <span class="period-label">{{ preset.label }}</span>
              <span class="period-range">{{ preset.rangeLabel }}</span>
            </button>
          </div>
          <div ref="timelineAxisRef" class="timeline-scroll" @scroll.passive="onTimelineScroll">
            <div class="timeline-slots">
              <div
                v-for="slot in timeSlots"
                :key="slot"
                class="timeline-row"
                :class="{
                  selected: isSlotSelected(slot),
                  'timeline-row--in-shift': slotInWorkBand(slot),
                  'timeline-row--shift-start': slot === shiftStartSlotAnchored,
                  'timeline-row--leave': daySchedule?.mode === 'leave',
                }"
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
          </div>
        </div>
      </section>
    </main>

    <van-popup v-model:show="showCreate" position="bottom" round :style="{ height: '64%' }">
      <div class="modal-body">
        <h3>新建事项</h3>
        <van-field v-model="draft.title" label="事项" placeholder="输入事项标题" />
        <van-field v-model="draft.start" label="开始" placeholder="例如 09:00" />
        <van-field v-model="draft.end" label="结束" placeholder="例如 09:30" />
        <van-field v-model="draft.notes" label="备注" placeholder="输入备注或语音转文字" type="textarea" rows="2" autosize />

        <div v-for="sec in scheduleQuickSections" :key="sec.sectionId" class="quick-section">
          <span class="quick-section-label">{{ sec.label }}</span>
          <div v-if="sec.items.length" class="quick-row quick-row--wrap">
            <van-button
              v-for="item in sec.items"
              :key="item.slug"
              size="small"
              type="primary"
              plain
              class="quick-pick-btn"
              @click="applyCatalogItem(item)"
            >
              {{ item.title }}
            </van-button>
          </div>
          <p v-else class="quick-empty-hint">暂无选项 — 可由管理员在「我的 → 行程快捷配置」中添加</p>
        </div>

        <div class="voice-actions">
          <van-button size="small" type="primary" plain @click="speechToText">语音转文字</van-button>
          <van-button size="small" type="primary" @click="voiceStructuredInput">语音结构化录入</van-button>
        </div>

        <van-button type="primary" block style="margin-top: 10px" @click="saveEvent">保存事项</van-button>
      </div>
    </van-popup>

    <van-calendar
      v-model:show="showMonthCalendar"
      :min-date="minCalendarDate"
      :max-date="maxCalendarDate"
      :default-date="calendarDefaultDate"
      @confirm="onCalendarConfirm"
    />

    <DailySchedulePicker
      v-model="showSchedulePicker"
      :date-str="selectedDate"
      :user-id="Number(user.id) || 1"
      :resolved-schedule="daySchedule"
      @applied="onScheduleApplied"
    />

    <AppBottomNav />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { showFailToast, showLoadingToast, showSuccessToast, closeToast } from 'vant'
import AppBottomNav from '../components/AppBottomNav.vue'
import DailySchedulePicker from '../components/DailySchedulePicker.vue'
import { getAttendanceByDate, getScheduleDay, getScheduleQuick, recognizeVoice, submitStatus } from '../api'
import { REGION_HOTEL_TITLE } from '../composables/useLocationCatalog'
import { STATUS_LABEL, normalizeAttendanceRecord } from '../data/models'

const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const records = ref([])
const selectedDate = ref(new Date().toISOString().slice(0, 10))
const showCreate = ref(false)
const selectionStart = ref('')
const selectionEnd = ref('')

const draft = reactive({
  title: '',
  start: '',
  end: '',
  notes: '',
  attStatus: 'OFFICE',
  attLocation: '行程',
})

const daySchedule = ref(null)
const showSchedulePicker = ref(false)

const loadDaySchedule = async () => {
  try {
    const { data } = await getScheduleDay(user.id || 1, selectedDate.value)
    daySchedule.value = data
  } catch {
    daySchedule.value = null
  }
}

/** 与后台 pillText / shift_types 一致展示；兼容旧接口「上班 …」前缀 */
const schedulePillLabel = computed(() => {
  const ds = daySchedule.value
  if (!ds) return '常规班（点击设置）'
  let text = String(ds.pillText || '').trim()
  if (text.startsWith('上班 ') && ds.mode === 'work' && ds.startTime && ds.endTime) {
    const nm = ds.label || ds.fullName || '班次'
    text = `常规班-${nm} ${ds.startTime}–${ds.endTime}`
  }
  if (text) return text
  if (ds.mode === 'leave') {
    const code = ds.code || ''
    const tail = [ds.description, ds.fullName].filter(Boolean).join(' ').trim()
    return tail ? `休假-${code} ${tail}` : `休假-${code}`
  }
  if (ds.mode === 'work' && ds.startTime && ds.endTime) {
    const nm = ds.label || ds.fullName || '班次'
    return `常规班-${nm} ${ds.startTime}–${ds.endTime}`
  }
  return '常规班（点击设置）'
})

const scheduleQuickSections = ref([])
const normalizeScheduleItem = (row) => ({
  slug: row.slug,
  category: row.category,
  region: row.region ?? '',
  title: row.title,
  subtitle: row.subtitle ?? '',
  detail: row.detail ?? '',
  sortOrder: row.sortOrder ?? row.sort_order ?? 0,
})

const loadScheduleQuick = async () => {
  try {
    const { data } = await getScheduleQuick()
    const raw = Array.isArray(data.sections) ? data.sections : []
    scheduleQuickSections.value = raw.map((sec) => ({
      sectionId: sec.sectionId ?? sec.section_id ?? sec.id,
      label: sec.label ?? '',
      sortOrder: sec.sortOrder ?? sec.sort_order ?? 0,
      itemCategory: sec.itemCategory ?? sec.item_category,
      itemRegion: sec.itemRegion ?? sec.item_region ?? '',
      items: Array.isArray(sec.items) ? sec.items.map(normalizeScheduleItem) : [],
    }))
  } catch {
    scheduleQuickSections.value = []
    showFailToast('快捷事项加载失败')
  }
}

const applyCatalogItem = (item) => {
  const cat = item.category
  if (cat === 'hotel_intro') {
    draft.attStatus = 'OFFICE'
    draft.attLocation = item.title
    draft.title = `在岗办公 · ${item.title}`
    if (!draft.notes) draft.notes = draft.title
    return
  }
  if (cat === 'dining_restaurant') {
    draft.attStatus = 'DINING'
    const hotelTitle = REGION_HOTEL_TITLE[item.region] || ''
    draft.attLocation = `${hotelTitle}·${item.title}`
    draft.title = `商务用餐 · ${item.title}`
    const sub = item.subtitle ? `${item.subtitle} · ${item.title}` : item.title
    if (!draft.notes) draft.notes = sub
    return
  }
  if (cat === 'schedule_chip') {
    draft.attStatus = 'OFFICE'
    draft.attLocation = '行程'
    draft.title = item.title
    if (!draft.notes) draft.notes = item.title
  }
}
/** 时间轴展示与「保存事项」唯一数据源：服务端 attendance_records（getAttendanceByDate / submitStatus）。 */
const timelineAxisRef = ref(null)
const showMonthCalendar = ref(false)

const minCalendarDate = new Date(new Date().getFullYear() - 1, 0, 1)
const maxCalendarDate = new Date(new Date().getFullYear() + 1, 11, 31)

const toLocalYMD = (date) => {
  const y = date.getFullYear()
  const mo = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${y}-${mo}-${day}`
}

const periodPresetsLive = computed(() => {
  const ds = daySchedule.value
  if (!ds) {
    return [
      { key: 'morning', label: '早', rangeLabel: '0–9', anchorSlot: '00:00' },
      { key: 'mid', label: '中', rangeLabel: '9–18', anchorSlot: '09:00' },
      { key: 'evening', label: '晚', rangeLabel: '18–24', anchorSlot: '18:00' },
    ]
  }
  if (ds.mode === 'leave') {
    return [
      { key: 'morning', label: '休', rangeLabel: '假别', anchorSlot: '09:00' },
      { key: 'mid', label: '全日', rangeLabel: '弹性', anchorSlot: '12:00' },
      { key: 'evening', label: '备忘', rangeLabel: '可选', anchorSlot: '18:00' },
    ]
  }
  const sm = slotToMinutes(ds.startTime)
  const em = slotToMinutes(ds.endTime)
  const preH = Math.max(0, Math.floor(sm / 60))
  const postH = Math.min(24, Math.ceil(em / 60))
  const anchorMid = minutesToSlot(sm - (sm % 30))
  const anchorEnd = minutesToSlot(Math.min(em - (em % 30), 23 * 60 + 30))
  return [
    { key: 'morning', label: '班前', rangeLabel: `0–${preH}`, anchorSlot: '00:00' },
    { key: 'mid', label: '班次', rangeLabel: `${ds.startTime}–${ds.endTime}`, anchorSlot: anchorMid },
    { key: 'evening', label: '班后', rangeLabel: `${postH}–24`, anchorSlot: anchorEnd },
  ]
})

const activePeriod = ref('mid')

const slotToMinutes = (slot) => {
  if (slot == null || typeof slot !== 'string') return 0
  const parts = slot.split(':')
  const h = Number(parts[0])
  const m = Number(parts[1])
  if (!Number.isFinite(h) || !Number.isFinite(m)) return 0
  return h * 60 + m
}

const periodKeyForSlotLive = (slot) => {
  const ds = daySchedule.value
  if (!ds) {
    const mins = slotToMinutes(slot)
    if (mins < 9 * 60) return 'morning'
    if (mins < 18 * 60) return 'mid'
    return 'evening'
  }
  if (ds.mode === 'leave') return 'mid'
  const sm = slotToMinutes(ds.startTime)
  const em = slotToMinutes(ds.endTime)
  const m = slotToMinutes(slot)
  if (m < sm) return 'morning'
  if (m < em) return 'mid'
  return 'evening'
}

const slotInWorkBand = (slot) => {
  const ds = daySchedule.value
  if (!ds || ds.mode !== 'work' || !ds.startTime || !ds.endTime) return false
  const sm = slotToMinutes(ds.startTime)
  const em = slotToMinutes(ds.endTime)
  const m = slotToMinutes(slot)
  return m >= sm && m < em
}

const defaultDayStartSlot = () => {
  const ds = daySchedule.value
  if (ds?.mode === 'work' && ds.startTime) return ds.startTime
  return '09:00'
}

const minutesToSlot = (totalMins) => {
  const maxMins = 23 * 60 + 30
  const clamped = Math.min(Math.max(totalMins, 0), maxMins)
  const h = Math.floor(clamped / 60)
  const m = clamped % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

/** 与 timeline-row data-slot 对齐的班次开始刻度（30 分钟网格） */
const shiftStartSlotAnchored = computed(() => {
  const ds = daySchedule.value
  if (!ds || ds.mode !== 'work' || !ds.startTime) return ''
  const sm = slotToMinutes(ds.startTime)
  const snapped = sm - (sm % 30)
  return minutesToSlot(snapped)
})

const slotPlusMinutes = (slot, deltaMins) => minutesToSlot(slotToMinutes(slot) + deltaMins)

let scrollSyncRaf = 0
const syncActivePeriodFromScroll = () => {
  const el = timelineAxisRef.value
  if (!el) return
  const rows = el.querySelectorAll('.timeline-slots .timeline-row[data-slot]')
  if (!rows.length) return
  const containerRect = el.getBoundingClientRect()
  const anchorY = containerRect.top + 12
  let bestSlot = null
  let bestDelta = Infinity
  rows.forEach((row) => {
    const slot = row.getAttribute('data-slot')
    if (!slot) return
    const top = row.getBoundingClientRect().top
    const delta = Math.abs(top - anchorY)
    if (delta < bestDelta) {
      bestDelta = delta
      bestSlot = slot
    }
  })
  if (bestSlot) activePeriod.value = periodKeyForSlotLive(bestSlot)
}

/** 进入页面 / 换日 / 换班后，将滚动区对齐到当日常规班开始刻度 */
const scrollTimelineToScheduleAnchor = async () => {
  await nextTick()
  const scroller = timelineAxisRef.value
  if (!scroller) return
  const anchor = shiftStartSlotAnchored.value || '09:00'
  const row = scroller.querySelector(`.timeline-slots .timeline-row[data-slot="${anchor}"]`)
  row?.scrollIntoView({ block: 'start', behavior: 'auto' })
  syncActivePeriodFromScroll()
}

const onScheduleApplied = async (payload) => {
  daySchedule.value = payload
  await scrollTimelineToScheduleAnchor()
}

const onTimelineScroll = () => {
  if (scrollSyncRaf) cancelAnimationFrame(scrollSyncRaf)
  scrollSyncRaf = requestAnimationFrame(() => {
    scrollSyncRaf = 0
    syncActivePeriodFromScroll()
  })
}

const jumpToPeriod = (key) => {
  const preset = periodPresetsLive.value.find((item) => item.key === key)
  const scroller = timelineAxisRef.value
  if (!preset || !scroller) return
  activePeriod.value = key
  const row = scroller.querySelector(`.timeline-slots .timeline-row[data-slot="${preset.anchorSlot}"]`)
  row?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const initPeriodFromClock = () => {
  const todayStr = toLocalYMD(new Date())
  if (selectedDate.value !== todayStr) {
    activePeriod.value = 'mid'
    return
  }
  const now = new Date()
  const hh = String(now.getHours()).padStart(2, '0')
  const half = Math.floor(now.getMinutes() / 30) * 30
  const slot = `${hh}:${String(half).padStart(2, '0')}`
  activePeriod.value = periodKeyForSlotLive(slot)
}

const formattedYmd = computed(() => {
  const d = new Date(`${selectedDate.value}T12:00:00`)
  const dow = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'][d.getDay()]
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日 ${dow}`
})

const calendarDefaultDate = computed(() => new Date(`${selectedDate.value}T12:00:00`))

const weekStartStr = computed(() => {
  const d = new Date(`${selectedDate.value}T12:00:00`)
  const day = d.getDay()
  const shift = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + shift)
  return toLocalYMD(d)
})

const weekDays = computed(() => {
  const base = new Date(`${weekStartStr.value}T12:00:00`)
  const todayStr = toLocalYMD(new Date())
  const short = ['一', '二', '三', '四', '五', '六', '日']
  return Array.from({ length: 7 }).map((_, index) => {
    const d = new Date(base)
    d.setDate(base.getDate() + index)
    const dateStr = toLocalYMD(d)
    return {
      date: dateStr,
      weekdayShort: short[index],
      dayNum: String(d.getDate()),
      isToday: dateStr === todayStr,
      isSelected: dateStr === selectedDate.value,
    }
  })
})

const timeSlots = computed(() => {
  const slots = []
  for (let hour = 0; hour <= 23; hour += 1) {
    for (let min = 0; min < 60; min += 30) {
      const slot = `${String(hour).padStart(2, '0')}:${String(min).padStart(2, '0')}`
      slots.push(slot)
    }
  }
  return slots
})

const selectedDateTimeSlot = computed(() => {
  const todayStr = toLocalYMD(new Date())
  const ds = daySchedule.value
  if (selectedDate.value !== todayStr) {
    return ds?.mode === 'work' && ds.startTime ? ds.startTime : '09:00'
  }
  const now = new Date()
  const hour = String(now.getHours()).padStart(2, '0')
  const half = Math.floor(now.getMinutes() / 30) * 30
  return `${hour}:${String(half).padStart(2, '0')}`
})

const sortedSummary = computed(() => {
  const counter = {}
  records.value.forEach((row) => { counter[row.status] = (counter[row.status] || 0) + 1 })
  return Object.entries(counter).map(([status, count]) => ({ key: status, label: STATUS_LABEL[status] || status, count })).sort((a, b) => b.count - a.count)
})

const loadRecords = async () => {
  try {
    const response = await getAttendanceByDate(user.id || 1, selectedDate.value)
    records.value = (response.data.records || []).map(normalizeAttendanceRecord)
  } catch {
    records.value = []
    showFailToast('行程数据加载失败')
  }
}

const applyDateChange = async () => {
  await loadRecords()
  await loadDaySchedule()
  await scrollTimelineToScheduleAnchor()
}

const selectScheduleDay = async (dateStr) => {
  selectedDate.value = dateStr
  await applyDateChange()
}

const shiftWeek = async (weekDelta) => {
  const d = new Date(`${selectedDate.value}T12:00:00`)
  d.setDate(d.getDate() + weekDelta * 7)
  selectedDate.value = toLocalYMD(d)
  await applyDateChange()
}

const onCalendarConfirm = async (value) => {
  const d = value instanceof Date ? value : new Date(value)
  selectedDate.value = toLocalYMD(d)
  showMonthCalendar.value = false
  await applyDateChange()
}

const openCreateModal = (slot = '') => {
  if (slot) {
    selectionStart.value = slot
    selectionEnd.value = slot
  }
  draft.title = ''
  draft.notes = ''
  draft.attStatus = 'OFFICE'
  draft.attLocation = '行程'
  draft.start = slot || selectionStart.value || defaultDayStartSlot()
  draft.end = slot ? slotPlusMinutes(slot, 30) : selectionEnd.value || slotPlusMinutes(draft.start, 30) || '09:30'
  showCreate.value = true
  loadScheduleQuick()
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

const saveEvent = async () => {
  const titleText = (draft.title || '').trim() || '行程事项'
  const notesText = (draft.notes || '').trim()
  const reasonParts = [titleText]
  const pill = daySchedule.value?.pillText
  if (pill) reasonParts.unshift(`【排班·${pill}】`)
  if (notesText) reasonParts.push(notesText)
  const defaultEnd = slotPlusMinutes(draft.start || '09:00', 30)
  if ((draft.end || '').trim() && draft.end.trim() !== defaultEnd) {
    reasonParts.push(`至 ${draft.end.trim()}`)
  }
  try {
    await submitStatus({
      userId: user.id || 1,
      status: draft.attStatus || 'OFFICE',
      location: (draft.attLocation || '').trim() || '行程',
      reason: reasonParts.join(' · '),
      occurredAt: `${selectedDate.value}T${(draft.start || '09:00').trim()}:00`,
    })
    await loadRecords()
    showCreate.value = false
    showSuccessToast('事项已保存')
  } catch {
    showFailToast('保存失败')
  }
}

const eventTitleBySlot = (slot) => {
  const slotStart = slotToMinutes(slot)
  const slotEnd = slotStart + 30
  const dayStr = selectedDate.value
  let lastReason = ''
  records.value.forEach((row) => {
    if (!row.occurredAt) return
    const d = new Date(row.occurredAt)
    if (Number.isNaN(d.getTime())) return
    if (toLocalYMD(d) !== dayStr) return
    const mins = d.getHours() * 60 + d.getMinutes()
    if (mins >= slotStart && mins < slotEnd) {
      lastReason = row.reason || row.statusLabel || ''
    }
  })
  return lastReason
}

const isSlotSelected = (slot) => {
  if (!selectionStart.value || !selectionEnd.value) return false
  return slot >= selectionStart.value && slot <= selectionEnd.value
}

onMounted(async () => {
  initPeriodFromClock()
  loadScheduleQuick()
  await loadRecords()
  await loadDaySchedule()
  await scrollTimelineToScheduleAnchor()
})

onUnmounted(() => {
  if (scrollSyncRaf) cancelAnimationFrame(scrollSyncRaf)
})
</script>

<style scoped>
.page-shell { min-height: 100vh; background:#f8f9fa; }
.schedule-topbar {
  height:48px;
  border-bottom:1px solid #c3c6d7;
  background:#fff;
  display:flex;
  align-items:center;
  justify-content:center;
  padding:0 12px;
}
.topbar-title { margin:0; font-size:20px; font-weight:700; color:#0f172a; }
.content { padding: 10px 12px var(--app-nav-clearance); display:flex; flex-direction:column; gap:8px; }
.calendar-card { padding:10px 8px 12px; }
.calendar-toolbar {
  display:flex;
  align-items:center;
  justify-content:space-between;
  gap:8px;
  margin-bottom:10px;
}
.week-nav-btn {
  flex-shrink:0;
  width:40px;
  height:40px;
  border:1px solid #e2e8f0;
  border-radius:999px;
  background:#fff;
  color:#2563eb;
  display:flex;
  align-items:center;
  justify-content:center;
  cursor:pointer;
  -webkit-tap-highlight-color:transparent;
}
.week-nav-btn:active { background:#f1f5f9; }
.week-nav-btn .material-symbols-outlined {
  font-size:22px;
  line-height:1;
}
.ymd-button {
  flex:1;
  min-width:0;
  border:none;
  border-radius:12px;
  background:#eff6ff;
  color:#1e40af;
  font-size:14px;
  font-weight:700;
  padding:10px 8px;
  cursor:pointer;
  line-height:1.35;
  -webkit-tap-highlight-color:transparent;
}
.ymd-button:active { filter:brightness(0.97); }
.week-strip {
  display:grid;
  grid-template-columns:repeat(7,minmax(0,1fr));
  gap:6px;
}
.week-day-cell {
  border:1px solid #e2e8f0;
  border-radius:12px;
  background:#fff;
  padding:8px 4px 10px;
  cursor:pointer;
  display:flex;
  flex-direction:column;
  align-items:center;
  gap:4px;
  color:#64748b;
  transition:border-color 0.15s ease, background 0.15s ease, box-shadow 0.15s ease;
  -webkit-tap-highlight-color:transparent;
}
.week-day-cell:active { transform:scale(0.98); }
.week-day-name { font-size:11px; font-weight:600; }
.week-day-num { font-size:17px; font-weight:800; color:#0f172a; line-height:1; }
.week-day-cell.today:not(.selected) {
  border-color:#fdba74;
  background:#fff7ed;
}
.week-day-cell.today:not(.selected) .week-day-num { color:#c2410c; }
.week-day-cell.selected {
  border-color:#2563eb;
  background:#eef4ff;
  color:#1d4ed8;
  box-shadow:0 0 0 2px rgba(37,99,235,0.15);
}
.week-day-cell.selected .week-day-name,
.week-day-cell.selected .week-day-num {
  color:#1d4ed8;
}
.week-day-cell.selected.today {
  border-color:#2563eb;
  background:linear-gradient(180deg,#eef4ff 0%,#fff7ed 100%);
}
.card { border:1px solid #c3c6d7; background:#fff; border-radius:10px; padding:10px; }
.card-title { margin:0 0 8px; font-size:15px; }
.timeline-card { padding-top:10px; }
.timeline-head-compact {
  display:flex;
  justify-content:space-between;
  align-items:center;
  gap:10px;
  margin-bottom:8px;
  flex-wrap:wrap;
}
.timeline-head-left {
  display:flex;
  align-items:center;
  gap:8px;
  min-width:0;
  flex:1;
}
.timeline-title { margin:0; font-size:15px; flex-shrink:0; }
.timeline-head-right {
  flex-shrink:0;
  display:flex;
  align-items:center;
  justify-content:flex-end;
}
.work-hours-pill {
  font-size:11px;
  font-weight:700;
  color:var(--brand-primary-mid,#2563eb);
  background:var(--brand-primary-soft,#eff6ff);
  border:1px solid #bfdbfe;
  border-radius:999px;
  padding:4px 10px;
  max-width:min(420px,52vw);
  min-width:0;
  flex:1;
  overflow:hidden;
  text-overflow:ellipsis;
  white-space:nowrap;
  cursor:pointer;
  font-family:inherit;
  -webkit-tap-highlight-color:transparent;
}
.work-hours-pill:active {
  filter:brightness(0.97);
}
.timeline-row--in-shift .slot-card {
  background:#f0fdf4;
  border-color:#86efac;
}
.timeline-row--leave .slot-card {
  opacity:0.92;
}
.timeline-row--shift-start .slot-time {
  font-weight:800;
  color:#1d4ed8;
}
.timeline-row--shift-start .slot-time::after {
  background:#2563eb;
}
.timeline-row--shift-start .slot-card {
  border-color:#93c5fd;
  box-shadow:inset 3px 0 0 #2563eb;
}
.summary-row {
  display:flex;
  align-items:center;
  gap:8px;
  margin-bottom:8px;
  min-height:28px;
}
.summary-row-label {
  flex-shrink:0;
  font-size:11px;
  font-weight:700;
  color:#64748b;
}
.summary-chips {
  display:flex;
  flex-wrap:wrap;
  gap:6px;
  min-width:0;
}
.summary-chip {
  font-size:11px;
  font-weight:600;
  color:#0f172a;
  background:#f1f5f9;
  border:1px solid #e2e8f0;
  border-radius:999px;
  padding:3px 8px;
}
.summary-empty { font-size:12px; color:#94a3b8; margin-bottom:8px; }
.timeline-actions { display:flex; gap:6px; flex-shrink:0; flex-wrap:wrap; justify-content:flex-end; }
.timeline-body {
  min-height:0;
}
.period-tabs {
  display:flex;
  gap:6px;
  margin-bottom:8px;
}
.period-tab {
  flex:1;
  min-width:0;
  margin:0;
  padding:8px 4px;
  border:1px solid #e2e8f0;
  border-radius:12px;
  background:#f8fafc;
  cursor:pointer;
  display:flex;
  flex-direction:column;
  align-items:center;
  justify-content:center;
  gap:4px;
  color:#475569;
  transition:background 0.15s ease, color 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
  -webkit-tap-highlight-color:transparent;
}
.period-tab:active {
  transform:scale(0.99);
}
.period-tab.active {
  background:#e8efff;
  border-color:#2563eb;
  color:#1d4ed8;
  box-shadow:0 0 0 2px rgba(37,99,235,0.2);
}
.period-label {
  font-size:16px;
  font-weight:800;
  line-height:1.1;
  letter-spacing:0.04em;
}
.period-range {
  font-size:11px;
  color:#64748b;
  font-weight:600;
  white-space:nowrap;
}
.period-tab.active .period-range {
  color:#2563eb;
}
.timeline-scroll {
  max-height:min(480px,calc(100vh - 200px));
  overflow-y:auto;
  overflow-x:hidden;
  scroll-behavior:smooth;
  border-radius:12px;
  border:1px solid #e2e8f0;
  background:#fafbfd;
}
.timeline-slots {
  display:flex;
  flex-direction:column;
  gap:6px;
  padding:6px 6px 8px 8px;
  background:#fff;
}
.timeline-row { display:grid; grid-template-columns:56px minmax(0,1fr); gap:6px; align-items:start; cursor:pointer; }
.timeline-row.selected .slot-card { border-color:#2563eb; background:#eef4ff; }
.slot-time { font-size:11px; color:#64748b; padding-top:8px; position:relative; }
.slot-time::after { content:''; position:absolute; left:48px; top:14px; width:6px; height:6px; border-radius:999px; background:#cbd5e1; }
.slot-card { border:1px solid #e2e8f0; border-radius:8px; min-height:44px; padding:6px 8px; background:#fff; }
.slot-text { display:block; font-size:12px; color:#0f172a; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
.slot-placeholder { font-size:12px; color:#94a3b8; }
.modal-body { padding:16px; }
.modal-body h3 { margin:0 0 10px; }
.quick-section { margin-top: 10px; }
.quick-section-label {
  display: block;
  font-size: 11px;
  font-weight: 700;
  color: #64748b;
  margin-bottom: 6px;
}
.quick-row {
  display: flex;
  flex-wrap: nowrap;
  gap: 6px;
}
.quick-row--wrap {
  flex-wrap: wrap;
}
.quick-pick-btn { margin: 0; }
.quick-empty-hint {
  margin: 0;
  font-size: 11px;
  color: #94a3b8;
  line-height: 1.35;
}
.voice-actions { display:flex; gap:8px; margin-top:10px; }
</style>
