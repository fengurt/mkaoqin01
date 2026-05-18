<template>
  <div class="page-shell page-shell--metro">
    <header class="metro-top">
      <div>
        <p class="metro-greet">{{ greetingText }}</p>
        <time class="metro-date" :datetime="todayText()">{{ todayTextDisplay }}</time>
      </div>
      <nav class="metro-top-links">
        <button type="button" class="metro-link ghost" @click="$router.push('/schedule')">行程</button>
        <button type="button" class="metro-link ghost" @click="$router.push('/my-attendance/day')">考勤</button>
      </nav>
    </header>

    <main class="metro-main" :class="{ 'metro-main--muted': homeBootstrapping }">
      <section class="metro-live" aria-label="主磁贴">
        <div class="metro-grid">
          <button
            type="button"
            class="metro-tile metro-tile--in"
            :disabled="checkInBusy"
            @click="handleCheckIn"
          >
            <span class="metro-tile-ico material-symbols-outlined" aria-hidden="true">login</span>
            <span class="metro-tile-label">一键打卡上班</span>
            <span v-if="checkInBusy" class="metro-tile-live">提交中…</span>
            <span v-else-if="hasCheckedInToday" class="metro-tile-live">今日已签到</span>
            <span v-else class="metro-tile-live">点击记录到岗</span>
          </button>
          <button
            type="button"
            class="metro-tile metro-tile--out"
            :disabled="clockOutBusy"
            :aria-busy="clockOutBusy"
            @click="handleClockOut"
          >
            <span class="metro-tile-ico material-symbols-outlined" aria-hidden="true">logout</span>
            <span class="metro-tile-label">一键快乐下班</span>
            <van-loading v-if="clockOutBusy" class="metro-tile-spinner" type="spinner" size="20px" color="#fff" />
            <span v-else-if="clockOutBlockedHint" class="metro-tile-live">{{ clockOutBlockedHint }}</span>
            <span v-else class="metro-tile-live">今日流程已完成？</span>
          </button>
          <button
            type="button"
            class="metro-tile metro-tile--report metro-tile--wide"
            :disabled="reportBusy"
            @click="openReportModal"
          >
            <span class="metro-tile-ico material-symbols-outlined" aria-hidden="true">bolt</span>
            <span class="metro-tile-label">快捷申报</span>
            <span class="metro-tile-live">{{ reportSubtitle }}</span>
          </button>
          <button type="button" class="metro-tile metro-tile--lead" @click="goLeads">
            <span class="metro-tile-ico material-symbols-outlined" aria-hidden="true">contact_phone</span>
            <span class="metro-tile-label">{{ t('home.leadTileTitle') }}</span>
            <span class="metro-tile-live">{{ leadSubtitle }}</span>
            <p v-if="leadPreview" class="metro-lead-name">{{ leadPreview.clientName }}</p>
            <div v-if="leadPreview" class="metro-lead-chips" aria-label="线索摘要">
              <span v-if="leadPreview.leadSegment" class="metro-lead-chip">{{ t(`lead.segment.${leadPreview.leadSegment}`) }}</span>
              <span v-if="leadPreview.approxOriginRegion" class="metro-lead-chip metro-lead-chip--muted">{{
                t(`lead.origin.${leadPreview.approxOriginRegion}`)
              }}</span>
              <span v-if="leadPreview.preferredVenue" class="metro-lead-chip metro-lead-chip--gold">{{
                t(`lead.venue.${leadPreview.preferredVenue}`)
              }}</span>
              <span v-if="leadPotentialComposite != null" class="metro-lead-chip metro-lead-chip--score"
                >{{ t('lead.radar.compositeShort') }} {{ leadPotentialComposite }}</span
              >
            </div>
          </button>
          <button type="button" class="metro-tile metro-tile--luck" @click="openFortunePoster">
            <span class="metro-tile-ico material-symbols-outlined" aria-hidden="true">auto_awesome</span>
            <span class="metro-tile-label">今日好运</span>
            <p class="metro-fortune">{{ fortuneLine }}</p>
          </button>
        </div>
      </section>
    </main>

    <van-popup v-model:show="showReportModal" position="bottom" round :style="{ height: '72%' }">
      <div class="report-modal">
        <div class="report-modal-head">
          <h3>工作申报</h3>
          <button type="button" class="modal-close" @click="showReportModal = false">关闭</button>
        </div>
        <div class="report-meta-card">
          <van-field
            v-model="form.dateDisplay"
            is-link
            readonly
            class="report-meta-field"
            label="日期"
            placeholder="选择日期"
            @click="showCalendar = true"
          />
          <div class="report-schedule-row report-meta-field">
            <span class="report-schedule-label">当日排班</span>
            <button type="button" class="report-schedule-pill" @click="showReportSchedulePicker = true">
              {{ reportSchedule?.pillText || '点击设置' }}
            </button>
          </div>
          <van-field v-model="form.location" class="report-meta-field report-meta-field--last" label="地点" placeholder="拜访/会议等默认地点（可选）" />
        </div>
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
                @click="selectReportTag(line, index)"
              >
                {{ tag.label }}
              </button>
            </div>
            <div v-if="tagOptions[line.tagIndex]?.introSlug" class="intro-link-row">
              <button type="button" class="text-link" @click="openIntro(tagOptions[line.tagIndex].introSlug)">
                查看地点介绍
              </button>
            </div>
            <div v-if="tagOptions[line.tagIndex]?.requiresDiningPick" class="dining-pick">
              <p class="dining-pick-label">澳门美高梅（半岛店）</p>
              <div class="dining-chip-grid">
                <div v-for="rest in peninsulaDining" :key="rest.slug" class="dining-chip-row">
                  <button
                    type="button"
                    class="dining-chip"
                    :class="{ active: line.diningSlug === rest.slug }"
                    @click="line.diningSlug = rest.slug"
                  >
                    {{ rest.title }}
                  </button>
                  <button type="button" class="dining-info-btn" @click="openIntro(rest.slug)">介绍</button>
                </div>
              </div>
              <p class="dining-pick-label dining-pick-label--spaced">美狮美高梅（氹仔店）</p>
              <div class="dining-chip-grid">
                <div v-for="rest in cotaiDining" :key="rest.slug" class="dining-chip-row">
                  <button
                    type="button"
                    class="dining-chip"
                    :class="{ active: line.diningSlug === rest.slug }"
                    @click="line.diningSlug = rest.slug"
                  >
                    {{ rest.title }}
                  </button>
                  <button type="button" class="dining-info-btn" @click="openIntro(rest.slug)">介绍</button>
                </div>
              </div>
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
        <van-button
          type="primary"
          block
          class="save-report-btn"
          :loading="reportBusy"
          loading-text="提交中..."
          @click="submitReport"
        >
          保存申报
        </van-button>
        <van-button
          v-if="reportSaved || hasReportedToday"
          type="success"
          block
          class="checkout-btn"
          :loading="clockOutBusy"
          loading-text="打卡中..."
          @click="submitClockOutFromModal"
        >
          打卡下班
        </van-button>
      </div>
    </van-popup>

    <van-calendar v-model:show="showCalendar" :min-date="minCalendarDate" :max-date="maxCalendarDate" @confirm="onCalendarConfirm" />

    <CatalogIntroPopup v-model="introOpen" :title="introTitle" :detail="introDetail" />

    <DailySchedulePicker
      v-model="showReportSchedulePicker"
      :date-str="form.date"
      :user-id="Number(user.id) || 1"
      :resolved-schedule="reportSchedule"
      @applied="onReportScheduleApplied"
    />

    <FortuneCalendarPopup
      v-model="showFortuneCalendar"
      :user-id="Number(user.id) || 1"
      :initial-date="form.date"
      title="今日好运"
    />

    <AppBottomNav />
    <BadgeCelebration ref="badgeCelebrationRef" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { showConfirmDialog, showFailToast, showSuccessToast } from 'vant'
import { useRouter } from 'vue-router'
import AppBottomNav from '../components/AppBottomNav.vue'
import BadgeCelebration from '../components/BadgeCelebration.vue'
import CatalogIntroPopup from '../components/CatalogIntroPopup.vue'
import DailySchedulePicker from '../components/DailySchedulePicker.vue'
import FortuneCalendarPopup from '../components/FortuneCalendarPopup.vue'
import { getAttendanceByDate, getFortuneDay, getLeadFeed, getScheduleDay, submitStatus } from '../api'
import { computeLeadValueRadar } from '../lib/leadValuePotential'
import { REGION_HOTEL_TITLE, useLocationCatalog } from '../composables/useLocationCatalog'
import { normalizeAttendanceRecord } from '../data/models'

const REPORT_STATUSES = ['OFFICE', 'OUTING', 'DINING', 'BUSINESS_TRIP']

const { t } = useI18n()
const router = useRouter()

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
const homeBootstrapping = ref(true)
const reportBusy = ref(false)
const clockOutBusy = ref(false)
const checkInBusy = ref(false)
const showFortuneCalendar = ref(false)
const todayFortuneCaption = ref('')

const reportSchedule = ref(null)
const showReportSchedulePicker = ref(false)

const badgeCelebrationRef = ref(null)
const leadPreview = ref(null)

const leadSubtitle = computed(() => {
  if (!leadPreview.value) return t('home.leadTileEmpty')
  return `${t('home.leadCreated')}: ${(leadPreview.value.createdAt || '').slice(0, 16)}`
})

const leadPotentialComposite = computed(() => {
  if (!leadPreview.value) return null
  return computeLeadValueRadar(leadPreview.value).composite
})

const loadLeadPreview = async () => {
  try {
    const { data } = await getLeadFeed(user.id || 1)
    const items = data.items || []
    leadPreview.value = items[0] || null
  } catch {
    leadPreview.value = null
  }
}

const goLeads = () => {
  router.push('/leads')
}

const playNewBadges = (list) => {
  if (Array.isArray(list) && list.length > 0) {
    badgeCelebrationRef.value?.play(list)
  }
}

const loadReportSchedule = async () => {
  try {
    const { data } = await getScheduleDay(user.id || 1, form.date)
    reportSchedule.value = data
  } catch {
    reportSchedule.value = null
  }
}

const onReportScheduleApplied = (payload) => {
  reportSchedule.value = payload
}

const scheduleReasonPrefix = computed(() => {
  const pill = reportSchedule.value?.pillText
  return pill ? `【排班·${pill}】` : ''
})

const FORTUNE_LINES = [
  '稳住节奏，今天适合把关键事往前推一小步。',
  '沟通比猜测更省力，开口就有转机。',
  '小事认真收尾，会换来一整天的轻松。',
  '留一点空白时间，反而更容易抓住重点。',
  '今日宜专注：一次只做一件要事。',
  '把目标写清楚，执行会顺很多。',
  '适当起身走动，思路会自己找上门。',
  '对同事多点耐心，效率会反弹回来。',
]

const isSameLocalDay = (occurredAtIso, ymd) => {
  if (!occurredAtIso || !ymd) return false
  const d = new Date(occurredAtIso)
  if (Number.isNaN(d.getTime())) return false
  const yy = d.getFullYear()
  const mo = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${yy}-${mo}-${day}` === ymd
}

const fortuneLine = computed(() => {
  if (todayFortuneCaption.value) return todayFortuneCaption.value
  const seed = [...todayText()].reduce((acc, character) => acc + character.charCodeAt(0), 0)
  const uid = Number(user.id) || 0
  const index = (seed + uid) % FORTUNE_LINES.length
  return FORTUNE_LINES[index]
})

const loadTodayFortune = async () => {
  try {
    const { data } = await getFortuneDay(user.id || 1, form.date)
    todayFortuneCaption.value = data.caption || ''
  } catch {
    todayFortuneCaption.value = ''
  }
}

const openFortunePoster = () => {
  showFortuneCalendar.value = true
}

const tagOptions = [
  { status: 'OUTING', label: '拜访客户' },
  { status: 'DINING', label: '商务用餐', requiresDiningPick: true },
  { status: 'OFFICE', label: '会议' },
  { status: 'BUSINESS_TRIP', label: '出差' },
  {
    status: 'OFFICE',
    label: '在岗办公 · 半岛店',
    presetLocation: '澳门美高梅（半岛店）',
    introSlug: 'mgm_peninsula_hotel',
  },
  {
    status: 'OFFICE',
    label: '在岗办公 · 氹仔店',
    presetLocation: '美狮美高梅（氹仔店）',
    introSlug: 'mgm_cotai_hotel',
  },
]

const { catalogBySlug, peninsulaDining, cotaiDining, loadLocationCatalog } = useLocationCatalog()

const introOpen = ref(false)
const introTitle = ref('')
const introDetail = ref('')

const openIntro = (slug) => {
  const row = catalogBySlug.value.get(slug)
  introTitle.value = row?.title || ''
  introDetail.value = row?.detail || ''
  if (!introDetail.value.trim()) {
    showFailToast('暂无介绍内容')
    return
  }
  introOpen.value = true
}

const selectReportTag = (line, index) => {
  line.tagIndex = index
  const tag = tagOptions[index]
  if (!tag?.requiresDiningPick) line.diningSlug = ''
}

let reportLineSeq = 0
const createEmptyReportLine = () => {
  reportLineSeq += 1
  return { id: reportLineSeq, tagIndex: 4, reason: '', diningSlug: '' }
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
  try {
    const response = await getAttendanceByDate(user.id || 1, todayText())
    records.value = (response.data.records || []).map(normalizeAttendanceRecord)
  } catch {
    records.value = []
    showFailToast('今日记录加载失败')
  } finally {
    homeBootstrapping.value = false
  }
}

const hasReportedToday = computed(() =>
  records.value.some((row) => REPORT_STATUSES.includes(row.status)),
)

const hasCheckedInToday = computed(() =>
  records.value.some(
    (row) => row.status === 'CHECK_IN' && isSameLocalDay(row.occurredAt, todayText()),
  ),
)

const reportSubtitle = computed(() => {
  if (reportBusy.value) return '正在提交…'
  if (hasReportedToday.value) return '今日已有申报 · 可继续补充'
  return '多行申报 · MGM 在岗与餐厅'
})

const clockOutBlockedHint = computed(() => {
  if (clockOutBusy.value) return ''
  if (!hasReportedToday.value) return '请先完成当日申报'
  return '随时可以快乐下班'
})

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
  loadReportSchedule()
}

const submitReport = async () => {
  const payloadLines = reportLines.value
    .map((line) => ({
      tagIndex: line.tagIndex,
      reason: line.reason.trim(),
      diningSlug: (line.diningSlug || '').trim(),
    }))
    .filter((line) => line.reason.length >= 2)

  if (payloadLines.length === 0) {
    showFailToast('请至少填写一条申报内容（每条至少2个字）')
    return
  }

  for (let i = 0; i < payloadLines.length; i += 1) {
    const line = payloadLines[i]
    const tag = tagOptions[line.tagIndex]
    if (tag?.requiresDiningPick) {
      if (!line.diningSlug) {
        showFailToast('商务用餐请选择餐厅')
        return
      }
      const row = catalogBySlug.value.get(line.diningSlug)
      if (!row) {
        showFailToast('餐厅目录未加载，请稍后重试')
        return
      }
    }
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

  const resolveLineLocation = (tag, line) => {
    if (tag?.presetLocation) return tag.presetLocation
    if (tag?.requiresDiningPick && line.diningSlug) {
      const row = catalogBySlug.value.get(line.diningSlug)
      if (row) {
        const hotelTitle = REGION_HOTEL_TITLE[row.region] || ''
        return `${hotelTitle}·${row.title}`
      }
    }
    return locationText
  }

  reportBusy.value = true
  try {
    const earnedBatch = []
    for (let i = 0; i < payloadLines.length; i += 1) {
      const line = payloadLines[i]
      const tag = tagOptions[line.tagIndex]
      const reasonText = [scheduleReasonPrefix.value, line.reason].filter(Boolean).join(' ').trim()
      const { data } = await submitStatus({
        userId: user.id || 1,
        status: tag.status,
        location: resolveLineLocation(tag, line),
        reason: reasonText,
        occurredAt: bumpOccurredAtLocal(baseOccurredAt, i),
      })
      if (Array.isArray(data?.newlyEarnedBadges)) {
        earnedBatch.push(...data.newlyEarnedBadges)
      }
    }
    reportSaved.value = true
    showSuccessToast(payloadLines.length > 1 ? `已保存 ${payloadLines.length} 条申报` : '申报已保存')
    await loadRecords()
    playNewBadges(earnedBatch)
  } catch {
    showFailToast('保存失败，请重试')
  } finally {
    reportBusy.value = false
  }
}

const submitClockOutFromModal = async () => {
  if (!hasReportedToday.value) {
    showFailToast('请先完成当日工作申报')
    return
  }
  clockOutBusy.value = true
  try {
    const { data } = await submitStatus({
      userId: user.id || 1,
      status: 'CHECK_OUT',
      location: form.location.trim() || '公司',
      reason: [scheduleReasonPrefix.value, '下班打卡'].filter(Boolean).join(' ').trim(),
      occurredAt: new Date().toISOString(),
    })
    showSuccessToast('下班打卡成功')
    showReportModal.value = false
    await loadRecords()
    playNewBadges(data?.newlyEarnedBadges)
  } catch {
    showFailToast('打卡失败，请重试')
  } finally {
    clockOutBusy.value = false
  }
}

const handleCheckIn = async () => {
  if (hasCheckedInToday.value) {
    showFailToast('今日已签到上班')
    return
  }
  await loadReportSchedule()
  const scheduleText = reportSchedule.value?.pillText || '未设置（将按默认班次记录）'
  try {
    await showConfirmDialog({
      title: '确认今日班次',
      message: `请确认当日排班：${scheduleText}`,
      confirmButtonText: '班次正确，继续打卡',
      cancelButtonText: '去调整班次',
    })
  } catch {
    showReportSchedulePicker.value = true
    return
  }

  const now = new Date()
  const currentMinutes = now.getHours() * 60 + now.getMinutes()
  const [startHour, startMinute] = String(reportSchedule.value?.startTime || '09:00').split(':').map(Number)
  const shiftStartMinutes = (Number.isFinite(startHour) ? startHour : 9) * 60 + (Number.isFinite(startMinute) ? startMinute : 0)
  const isLate = currentMinutes > shiftStartMinutes

  checkInBusy.value = true
  try {
    const { data } = await submitStatus({
      userId: user.id || 1,
      status: 'CHECK_IN',
      location: '公司',
      reason: [scheduleReasonPrefix.value, '上班打卡'].filter(Boolean).join(' ').trim(),
      occurredAt: new Date().toISOString(),
    })
    showSuccessToast(isLate ? '上班打卡成功（今日已迟到）' : '上班打卡成功（今日正常上班）')
    await loadRecords()
    playNewBadges(data?.newlyEarnedBadges)
  } catch {
    showFailToast('打卡失败，请重试')
  } finally {
    checkInBusy.value = false
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
  await loadReportSchedule()

  const now = new Date()
  const currentMinutes = now.getHours() * 60 + now.getMinutes()
  const [endHour, endMinute] = String(reportSchedule.value?.endTime || '18:00').split(':').map(Number)
  const shiftEndMinutes = (Number.isFinite(endHour) ? endHour : 18) * 60 + (Number.isFinite(endMinute) ? endMinute : 0)
  if (currentMinutes < shiftEndMinutes) {
    try {
      await showConfirmDialog({
        title: '提前下班确认',
        message: `当前未到下班时间（班次结束 ${reportSchedule.value?.endTime || '18:00'}），确认提前下班？`,
        confirmButtonText: '确认提前下班',
        cancelButtonText: '取消',
      })
    } catch {
      return
    }
  }

  clockOutBusy.value = true
  try {
    const { data } = await submitStatus({
      userId: user.id || 1,
      status: 'CHECK_OUT',
      location: '公司',
      reason: [scheduleReasonPrefix.value, '下班打卡'].filter(Boolean).join(' ').trim(),
      occurredAt: new Date().toISOString(),
    })
    showSuccessToast('下班打卡成功')
    await loadRecords()
    playNewBadges(data?.newlyEarnedBadges)
  } catch {
    showFailToast('打卡失败，请重试')
  } finally {
    clockOutBusy.value = false
  }
}

watch(showReportModal, (visible) => {
  if (visible) {
    reportSaved.value = hasReportedToday.value
    loadReportSchedule()
  }
})

watch(
  () => form.date,
  () => {
    if (showReportModal.value) loadReportSchedule()
  },
)

watch(() => form.date, () => {
  loadTodayFortune()
})

onMounted(async () => {
  loadLocationCatalog()
  await loadRecords()
  await loadReportSchedule()
  await loadLeadPreview()
  await loadTodayFortune()
})
</script>

<style scoped>
.page-shell--metro {
  min-height: 100vh;
  background: var(--brand-surface);
  color: var(--brand-title);
}
.metro-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  padding: 14px 16px 12px;
  background: var(--metro-header-bg);
  color: var(--metro-header-text);
  border-bottom: 2px solid var(--accent-gold, #b8954f);
  box-shadow: 0 12px 32px rgba(10, 12, 18, 0.35);
}
.metro-greet {
  margin: 0;
  font-size: 13px;
  opacity: 0.78;
}
.metro-date {
  display: block;
  margin-top: 2px;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 0.04em;
  font-variant-numeric: tabular-nums;
  font-family: "Noto Serif SC", "Songti SC", serif;
}
.metro-top-links {
  display: flex;
  gap: 6px;
}
.metro-link.ghost {
  border: 1px solid rgba(184, 149, 79, 0.45);
  background: rgba(255, 255, 255, 0.04);
  color: #f7f3eb;
  font-size: 12px;
  font-weight: 600;
  padding: 6px 11px;
  border-radius: 999px;
}
.metro-main {
  padding: 14px 14px calc(var(--app-nav-clearance) + 8px);
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.metro-main--muted {
  opacity: 0.55;
  pointer-events: none;
}
.metro-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.metro-tile {
  border: 0;
  border-radius: 16px;
  min-height: 104px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: flex-end;
  text-align: left;
  cursor: pointer;
  color: #fff;
  box-shadow: 0 10px 26px rgba(15, 18, 26, 0.18);
}
.metro-tile:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.metro-tile--wide {
  grid-column: 1 / -1;
  min-height: 118px;
}
.metro-tile--in {
  background: var(--metro-tile-in);
}
.metro-tile--out {
  background: var(--metro-tile-out);
}
.metro-tile--report {
  background: var(--metro-tile-report);
}
.metro-tile--luck {
  background: var(--metro-tile-luck);
  min-height: 120px;
}
.metro-tile--lead {
  min-height: 120px;
  background: var(--metro-tile-lead, linear-gradient(142deg, #1a1816 0%, #2a2622 42%, #3a3228 100%));
  border: 1px solid rgba(184, 149, 79, 0.35);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.06),
    0 12px 28px rgba(10, 10, 12, 0.45);
}
.metro-lead-name {
  margin: 8px 0 0;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.35;
  opacity: 0.96;
  text-align: left;
  max-height: 2.8em;
  overflow: hidden;
  font-family: "Noto Serif SC", "Songti SC", serif;
}
.metro-lead-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
.metro-lead-chip {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 4px 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: rgba(247, 243, 235, 0.95);
}
.metro-lead-chip--muted {
  text-transform: none;
  letter-spacing: 0;
  font-weight: 600;
  opacity: 0.88;
}
.metro-lead-chip--gold {
  background: rgba(184, 149, 79, 0.22);
  border-color: rgba(184, 149, 79, 0.55);
  color: #fdf6e4;
}
.metro-lead-chip--score {
  text-transform: none;
  letter-spacing: 0.02em;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  background: rgba(253, 246, 228, 0.35);
  border-color: rgba(253, 246, 228, 0.55);
}
.metro-tile-ico {
  font-size: 30px;
  opacity: 0.95;
  margin-bottom: auto;
}
.metro-tile-label {
  font-size: 17px;
  font-weight: 800;
  line-height: 1.25;
}
.metro-tile-live {
  margin-top: 5px;
  font-size: 12px;
  opacity: 0.9;
}
.metro-tile-spinner {
  margin-top: 6px;
}
.metro-fortune {
  margin: 6px 0 0;
  font-size: 14px;
  line-height: 1.45;
  font-weight: 500;
  opacity: 0.95;
  max-height: 3.15em;
  overflow: hidden;
}
.metro-pivot {
  background: var(--metro-pivot-bg);
  border-radius: 16px;
  border: 1px solid var(--brand-border);
  box-shadow: 0 4px 18px rgba(20, 24, 33, 0.05);
}
.metro-pivot-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 10px 12px 4px;
  font-size: 11px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--brand-subtext);
}
.metro-pivot-head small {
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
  color: var(--brand-subtext);
  opacity: 0.85;
}
.metro-pivot-track {
  padding: 0 8px 10px;
  overflow-x: auto;
  scroll-snap-type: x proximity;
  -webkit-overflow-scrolling: touch;
}
.metro-chips {
  display: flex;
  flex-wrap: nowrap;
  gap: 8px;
  padding: 8px 4px;
}
.metro-chip {
  flex: 0 0 auto;
  scroll-snap-align: start;
  font-size: 12px;
  font-weight: 700;
  background: var(--metro-chip-bg);
  color: var(--metro-header-text);
  padding: 8px 12px;
  border-radius: 8px;
}
.metro-cards {
  display: flex;
  flex-direction: row;
  gap: 10px;
  padding: 4px;
}
.metro-mini-card {
  flex: 0 0 auto;
  width: min(164px, 52vw);
  scroll-snap-align: start;
  background: var(--brand-card);
  border: 1px solid var(--brand-border);
  border-radius: 10px;
  padding: 10px;
}
.metro-mini-time {
  font-size: 11px;
  font-weight: 700;
  color: var(--brand-text);
  font-variant-numeric: tabular-nums;
}
.metro-mini-title {
  margin: 6px 0 4px;
  font-size: 14px;
  font-weight: 800;
  color: var(--brand-title);
}
.metro-mini-meta {
  margin: 0;
  font-size: 11px;
  color: var(--brand-subtext);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.metro-pivot-empty {
  margin: 0;
  padding: 12px 8px;
  font-size: 13px;
  color: var(--brand-subtext);
}
.today-shift-row {
  padding: 10px 12px 12px;
}
.fortune-modal {
  padding: 8px;
  background: #fff;
}
.fortune-image {
  width: 100%;
  border-radius: 10px;
  display: block;
}
.fortune-fallback {
  margin: 0;
  padding: 24px 14px;
  text-align: center;
  font-size: 14px;
  color: var(--brand-subtext);
}

.report-modal { padding:10px 12px 16px; max-height:100%; overflow:auto; }
.report-modal-head {
  position: sticky;
  top: 0;
  z-index: 1;
  display:flex;
  justify-content:space-between;
  align-items:center;
  margin-bottom:10px;
  padding: 2px 0 8px;
  background: #fff;
}
.report-modal-head h3 { margin:0; font-size:18px; color:#0f172a; }
.report-meta-card {
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #f8fbff;
  overflow: hidden;
  margin-bottom: 8px;
}
.report-meta-field {
  border-bottom: 1px solid #e8eef9;
}
.report-meta-field--last {
  border-bottom: 0;
}
.report-meta-field :deep(.van-field__label) {
  color: #64748b;
  font-weight: 600;
}
.report-schedule-row {
  display:flex;
  align-items:center;
  justify-content:space-between;
  gap:10px;
  padding:10px 12px;
}
.report-schedule-label {
  flex-shrink:0;
  font-size:13px;
  font-weight:600;
  color:#64748b;
}
.report-schedule-pill {
  font-size:11px;
  font-weight:700;
  color:var(--brand-primary-mid,#2563eb);
  background:var(--brand-primary-soft,#eff6ff);
  border:1px solid #bfdbfe;
  border-radius:999px;
  padding:6px 12px;
  white-space:nowrap;
  cursor:pointer;
  font-family:inherit;
  -webkit-tap-highlight-color:transparent;
}
.report-schedule-pill:active {
  filter:brightness(0.97);
}
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
.lines-section { margin-top:6px; }
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

.intro-link-row { margin: 2px 0 6px; }
.text-link {
  border: 0;
  background: transparent;
  padding: 0;
  font-size: 12px;
  color: #3156cb;
  text-decoration: underline;
}
.dining-pick { margin: 4px 0 8px; padding: 8px; border-radius: 8px; background: #fff; border: 1px dashed #c7d2fe; }
.dining-pick-label { margin: 0 0 6px; font-size: 11px; color: #64748b; font-weight: 600; }
.dining-pick-label--spaced { margin-top: 10px; }
.dining-chip-grid { display: flex; flex-direction: column; gap: 6px; }
.dining-chip-row { display: flex; align-items: center; gap: 6px; }
.dining-chip {
  flex: 1;
  min-width: 0;
  border: 1px solid #dbe3ef;
  border-radius: 999px;
  padding: 5px 10px;
  font-size: 11px;
  background: #fff;
  color: #475569;
  text-align: left;
}
.dining-chip.active {
  border-color: #3156cb;
  background: #eef4ff;
  color: #3156cb;
  font-weight: 700;
}
.dining-info-btn {
  flex-shrink: 0;
  border: 0;
  background: transparent;
  color: #94a3b8;
  font-size: 11px;
  padding: 4px 6px;
}
</style>
