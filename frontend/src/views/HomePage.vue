<template>
  <div class="page-shell">
    <header class="topbar">
      <div>
        <p class="welcome">{{ greetingText }}</p>
        <h1>首页</h1>
      </div>
      <button class="icon-btn" type="button">
        <span class="material-symbols-outlined">notifications</span>
      </button>
    </header>

    <main class="content">
      <section class="hero-row">
        <button type="button" class="hero-card hero-card-report" @click="openReportModal">
          <span class="material-symbols-outlined hero-icon">task</span>
          <div class="hero-text">
            <h2>工作申报</h2>
            <p>{{ todayTextDisplay }}</p>
          </div>
          <span class="material-symbols-outlined hero-chevron">chevron_right</span>
        </button>
        <button type="button" class="hero-card hero-card-out" @click="handleClockOut">
          <span class="material-symbols-outlined hero-icon">logout</span>
          <div class="hero-text">
            <h2>一键打卡下班</h2>
            <p v-if="hasReportedToday">今日已申报，可打卡</p>
            <p v-else>需先完成当日申报</p>
          </div>
          <span class="material-symbols-outlined hero-chevron">chevron_right</span>
        </button>
      </section>

      <section class="card">
        <h2 class="card-title">关键指标</h2>
        <div class="kpi-grid">
          <div class="kpi-item"><span>记录总数</span><strong>{{ records.length }}</strong></div>
          <div class="kpi-item"><span>最高频行为</span><strong>{{ topActionLabel }}</strong></div>
        </div>
      </section>

      <section class="card">
        <h2 class="card-title">快捷入口</h2>
        <div class="quick-grid">
          <button class="quick-btn" @click="$router.push('/schedule')"><span class="material-symbols-outlined">event_note</span><span>进入行程</span></button>
          <button class="quick-btn" @click="$router.push('/my-attendance/day')"><span class="material-symbols-outlined">calendar_month</span><span>我的考勤</span></button>
          <button class="quick-btn" @click="$router.push('/me')"><span class="material-symbols-outlined">person</span><span>我的主页</span></button>
          <button class="quick-btn" @click="$router.push('/employee/profile')"><span class="material-symbols-outlined">badge</span><span>个人资料</span></button>
        </div>
      </section>

      <section class="card">
        <h2 class="card-title">关键行为汇总（今日）</h2>
        <div v-if="sortedSummary.length" class="summary-list">
          <div v-for="item in sortedSummary" :key="item.key" class="summary-item">
            <span>{{ item.label }}</span>
            <strong>{{ item.count }}</strong>
          </div>
        </div>
        <div v-else class="empty">今天还没有行为记录，可使用上方「工作申报」或去行程新建。</div>
      </section>

      <section class="card">
        <h2 class="card-title">最近动态</h2>
        <div v-if="records.length === 0" class="empty">暂无行为记录</div>
        <div v-else class="timeline-list">
          <div v-for="record in records" :key="record.id" class="timeline-item">
            <div class="timeline-time">{{ formatClock(record.occurredAt) }}</div>
            <div class="timeline-content">
              <div class="timeline-title">{{ record.statusLabel }}</div>
              <div class="timeline-desc">{{ record.location }} · {{ record.reason }}</div>
            </div>
          </div>
        </div>
      </section>
    </main>

    <van-popup v-model:show="showReportModal" position="bottom" round :style="{ height: '72%' }">
      <div class="report-modal">
        <div class="report-modal-head">
          <h3>工作申报</h3>
          <button type="button" class="modal-close" @click="showReportModal = false">关闭</button>
        </div>
        <van-field
          v-model="form.dateDisplay"
          is-link
          readonly
          label="日期"
          placeholder="选择日期"
          @click="showCalendar = true"
        />
        <van-field v-model="form.location" label="地点" placeholder="默认公司" />
        <div class="lines-section">
          <div class="lines-head">
            <p class="tag-label">申报内容</p>
            <button type="button" class="add-line-btn" @click="addReportLine">新增一条</button>
          </div>
          <div v-for="(line, lineIndex) in reportLines" :key="line.id" class="report-line">
            <div class="line-toolbar">
              <span class="line-label">第 {{ lineIndex + 1 }} 条</span>
              <button
                v-if="reportLines.length > 1"
                type="button"
                class="remove-line-btn"
                @click="removeReportLine(lineIndex)"
              >
                删除
              </button>
            </div>
            <div class="tag-row tag-row-compact">
              <button
                v-for="(tag, index) in tagOptions"
                :key="`${tag.status}-${tag.label}-${line.id}`"
                type="button"
                class="tag-btn tag-btn-compact"
                :class="{ active: line.tagIndex === index }"
                @click="line.tagIndex = index"
              >
                {{ tag.label }}
              </button>
            </div>
            <van-field
              v-model="line.reason"
              type="textarea"
              placeholder="简要说明（至少2字）"
              rows="2"
              maxlength="240"
              show-word-limit
              class="compact-reason-field"
            />
          </div>
        </div>
        <van-button type="primary" block class="save-report-btn" @click="submitReport">保存申报</van-button>
        <van-button
          v-if="reportSaved || hasReportedToday"
          type="success"
          block
          class="checkout-btn"
          @click="submitClockOutFromModal"
        >
          打卡下班
        </van-button>
      </div>
    </van-popup>

    <van-calendar v-model:show="showCalendar" :min-date="minCalendarDate" :max-date="maxCalendarDate" @confirm="onCalendarConfirm" />

    <AppBottomNav current="home" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { showConfirmDialog, showFailToast, showSuccessToast } from 'vant'
import AppBottomNav from '../components/AppBottomNav.vue'
import { getAttendanceByDate, submitStatus } from '../api'
import { formatClock, normalizeAttendanceRecord, STATUS_LABEL } from '../data/models'

const REPORT_STATUSES = ['OFFICE', 'OUTING', 'DINING', 'BUSINESS_TRIP']

const user = JSON.parse(localStorage.getItem('user') || '{"id":1}')
const records = ref([])

const todayText = () => new Date().toISOString().slice(0, 10)
const todayTextDisplay = computed(() => todayText().replaceAll('-', '.'))
const greetingText = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return '早上好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

const showReportModal = ref(false)
const showCalendar = ref(false)
const reportSaved = ref(false)

const tagOptions = [
  { status: 'OUTING', label: '拜访客户' },
  { status: 'DINING', label: '商务用餐' },
  { status: 'OFFICE', label: '会议' },
  { status: 'BUSINESS_TRIP', label: '出差' },
  { status: 'OFFICE', label: '在岗办公' },
]

let reportLineSeq = 0
const createEmptyReportLine = () => {
  reportLineSeq += 1
  return { id: reportLineSeq, tagIndex: 4, reason: '' }
}
const reportLines = ref([createEmptyReportLine()])

const form = reactive({
  date: todayText(),
  dateDisplay: '',
  location: '公司',
})

const syncDateDisplay = () => {
  form.dateDisplay = form.date.replaceAll('-', '.')
}

syncDateDisplay()

const minCalendarDate = new Date(new Date().getFullYear() - 1, 0, 1)
const maxCalendarDate = new Date(new Date().getFullYear() + 1, 11, 31)

const onCalendarConfirm = (value) => {
  const d = value instanceof Date ? value : new Date(value)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  form.date = `${y}-${m}-${day}`
  syncDateDisplay()
  showCalendar.value = false
}

const loadRecords = async () => {
  const response = await getAttendanceByDate(user.id || 1, todayText())
  records.value = (response.data.records || []).map(normalizeAttendanceRecord)
}

const hasReportedToday = computed(() =>
  records.value.some((row) => REPORT_STATUSES.includes(row.status)),
)

const sortedSummary = computed(() => {
  const counter = {}
  records.value.forEach((row) => {
    counter[row.status] = (counter[row.status] || 0) + 1
  })
  return Object.entries(counter)
    .map(([status, count]) => ({ key: status, label: STATUS_LABEL[status] || status, count }))
    .sort((left, right) => right.count - left.count)
})
const topActionLabel = computed(() => sortedSummary.value[0]?.label || '无')

const occurredAtForFormDate = () => {
  const now = new Date()
  const hh = String(now.getHours()).padStart(2, '0')
  const mm = String(now.getMinutes()).padStart(2, '0')
  return `${form.date}T${hh}:${mm}:00`
}

const bumpOccurredAtLocal = (yyyyMmDdThhMmSs, offsetSeconds) => {
  const [datePart, timePart] = yyyyMmDdThhMmSs.split('T')
  const [y, mo, d] = datePart.split('-').map(Number)
  const [hh, mm, ss] = timePart.split(':').map(Number)
  const dt = new Date(y, mo - 1, d, hh, mm, (ss || 0) + offsetSeconds)
  const pad = (n) => String(n).padStart(2, '0')
  return `${dt.getFullYear()}-${pad(dt.getMonth() + 1)}-${pad(dt.getDate())}T${pad(dt.getHours())}:${pad(dt.getMinutes())}:${pad(dt.getSeconds())}`
}

const addReportLine = () => {
  reportLines.value.push(createEmptyReportLine())
}

const removeReportLine = (index) => {
  if (reportLines.value.length <= 1) return
  reportLines.value.splice(index, 1)
}

const openReportModal = () => {
  form.date = todayText()
  syncDateDisplay()
  form.location = '公司'
  reportLines.value = [createEmptyReportLine()]
  reportSaved.value = hasReportedToday.value
  showReportModal.value = true
}

const submitReport = async () => {
  const payloadLines = reportLines.value
    .map((line) => ({
      tagIndex: line.tagIndex,
      reason: line.reason.trim(),
    }))
    .filter((line) => line.reason.length >= 2)

  if (payloadLines.length === 0) {
    showFailToast('请至少填写一条申报内容（每条至少2个字）')
    return
  }

  try {
    await showConfirmDialog({
      title: '确认申报内容',
      message:
        '请确认是否已经填入当天主要行程信息，包括：餐饮用餐、拜访客户、外出记录、工作场点记录等。',
      confirmButtonText: '已确认，提交',
      cancelButtonText: '返回修改',
    })
  } catch {
    return
  }

  const locationText = form.location.trim() || '公司'
  const baseOccurredAt = occurredAtForFormDate()

  try {
    for (let i = 0; i < payloadLines.length; i += 1) {
      const line = payloadLines[i]
      const tag = tagOptions[line.tagIndex]
      await submitStatus({
        userId: user.id || 1,
        status: tag.status,
        location: locationText,
        reason: line.reason,
        occurredAt: bumpOccurredAtLocal(baseOccurredAt, i),
      })
    }
    reportSaved.value = true
    showSuccessToast(payloadLines.length > 1 ? `已保存 ${payloadLines.length} 条申报` : '申报已保存')
    await loadRecords()
  } catch {
    showFailToast('保存失败，请重试')
  }
}

const submitClockOutFromModal = async () => {
  if (!hasReportedToday.value) {
    showFailToast('请先完成当日工作申报')
    return
  }
  try {
    await submitStatus({
      userId: user.id || 1,
      status: 'CHECK_OUT',
      location: form.location.trim() || '公司',
      reason: '下班打卡',
      occurredAt: new Date().toISOString(),
    })
    showSuccessToast('下班打卡成功')
    showReportModal.value = false
    await loadRecords()
  } catch {
    showFailToast('打卡失败，请重试')
  }
}

const handleClockOut = async () => {
  if (!hasReportedToday.value) {
    try {
      await showConfirmDialog({
        title: '请完成当日申报',
        message: '需要先填写今日工作申报后才能打卡下班。',
        confirmButtonText: '去申报',
        cancelButtonText: '取消',
      })
      openReportModal()
    } catch {
      /* dismissed */
    }
    return
  }
  try {
    await submitStatus({
      userId: user.id || 1,
      status: 'CHECK_OUT',
      location: '公司',
      reason: '下班打卡',
      occurredAt: new Date().toISOString(),
    })
    showSuccessToast('下班打卡成功')
    await loadRecords()
  } catch {
    showFailToast('打卡失败，请重试')
  }
}

watch(showReportModal, (visible) => {
  if (visible) {
    reportSaved.value = hasReportedToday.value
  }
})

onMounted(loadRecords)
</script>

<style scoped>
.page-shell { min-height: 100vh; background:#f6f8ff; }
.topbar { height:72px; background:#fff; border-bottom:1px solid #d8e0f5; display:flex; align-items:center; justify-content:space-between; padding:0 16px; }
.welcome { margin:0; font-size:12px; color:#64748b; }
.topbar h1 { margin:0; font-size:22px; color:#0f172a; }
.icon-btn { width:38px; height:38px; border:1px solid #d8e0f5; border-radius:999px; background:#fff; color:#3156cb; }
.content { padding:16px 16px 84px; display:flex; flex-direction:column; gap:12px; }

.hero-row { display:flex; flex-direction:column; gap:10px; }
.hero-card {
  border:0;
  border-radius:14px;
  padding:14px;
  display:flex;
  align-items:center;
  gap:12px;
  text-align:left;
  cursor:pointer;
  box-shadow:0 12px 26px rgba(46,88,214,0.18);
  color:#fff;
}
.hero-card-report {
  background:linear-gradient(135deg,#2e58d6,#6b8dff);
}
.hero-card-out {
  background:linear-gradient(135deg,#0f766e,#14b8a6);
}
.hero-card h2 { margin:0; font-size:16px; }
.hero-card p { margin:4px 0 0; font-size:12px; opacity:0.92; }
.hero-icon { font-size:28px; opacity:0.95; }
.hero-text { flex:1; min-width:0; }
.hero-chevron { font-size:22px; opacity:0.85; }

.card { background:#fff; border:1px solid #d8e0f5; border-radius:12px; padding:12px; box-shadow:0 8px 18px rgba(15,40,120,0.05); }
.card-title { margin:0 0 10px; font-size:16px; color:#0f172a; }
.kpi-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.kpi-item { border:1px solid #e4eafc; border-radius:10px; padding:10px; display:flex; flex-direction:column; gap:6px; }
.kpi-item span { color:#64748b; font-size:12px; }
.kpi-item strong { color:#3156cb; font-size:16px; }
.summary-list { display:flex; flex-direction:column; gap:8px; }
.summary-item { display:flex; justify-content:space-between; border:1px solid #e4eafc; border-radius:10px; padding:8px 10px; font-size:13px; }
.summary-item strong { color:#3156cb; }
.empty { color:#64748b; font-size:13px; }
.timeline-list { display:flex; flex-direction:column; gap:8px; }
.timeline-item { display:flex; gap:10px; border-bottom:1px dashed #e4eafc; padding:8px 0; }
.timeline-time { font-size:12px; color:#64748b; width:60px; }
.timeline-title { font-size:13px; font-weight:700; }
.timeline-desc { font-size:12px; color:#64748b; margin-top:2px; }
.quick-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:8px; }
.quick-btn { border:1px solid #dbe3ef; border-radius:10px; background:#fff; padding:10px; display:flex; flex-direction:column; align-items:flex-start; gap:6px; color:#0f172a; font-size:13px; transition:all .2s; }
.quick-btn:active { transform:translateY(1px) scale(0.99); background:#f1f5ff; }
.quick-btn .material-symbols-outlined { color:#3156cb; }

.report-modal { padding:16px 16px 24px; max-height:100%; overflow:auto; }
.report-modal-head { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; }
.report-modal-head h3 { margin:0; font-size:18px; color:#0f172a; }
.modal-close { border:0; background:transparent; color:#3156cb; font-size:14px; }
.tag-section { margin:8px 0; }
.tag-label { margin:0 0 8px; font-size:12px; color:#64748b; }
.tag-row { display:flex; flex-wrap:wrap; gap:8px; }
.tag-btn {
  border:1px solid #dbe3ef;
  border-radius:999px;
  padding:6px 12px;
  font-size:12px;
  background:#fff;
  color:#475569;
}
.tag-btn.active {
  border-color:#3156cb;
  background:#eef4ff;
  color:#3156cb;
  font-weight:700;
}
.lines-section { margin-top:4px; }
.lines-head { display:flex; align-items:center; justify-content:space-between; gap:8px; margin-bottom:8px; }
.lines-head .tag-label { margin:0; }
.add-line-btn {
  border:1px solid #3156cb;
  border-radius:8px;
  padding:4px 10px;
  font-size:12px;
  background:#fff;
  color:#3156cb;
}
.report-line {
  border:1px solid #e4eafc;
  border-radius:10px;
  padding:8px 10px 4px;
  margin-bottom:10px;
  background:#fafbff;
}
.line-toolbar {
  display:flex;
  align-items:center;
  justify-content:space-between;
  margin-bottom:6px;
}
.line-label { font-size:12px; color:#64748b; }
.remove-line-btn {
  border:0;
  background:transparent;
  color:#94a3b8;
  font-size:12px;
}
.tag-row-compact { gap:6px; margin-bottom:4px; }
.tag-btn-compact { padding:4px 8px; font-size:11px; }
.compact-reason-field :deep(.van-field__control) {
  font-size:13px;
  line-height:1.35;
  min-height:44px;
}
.compact-reason-field :deep(.van-field__word-limit) {
  font-size:11px;
}
.save-report-btn { margin-top:12px; }
.checkout-btn { margin-top:10px; }
</style>
