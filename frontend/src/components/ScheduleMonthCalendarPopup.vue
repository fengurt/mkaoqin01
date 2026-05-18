<template>
  <van-popup v-model:show="visible" position="bottom" round :style="{ height: popupHeight }">
    <div class="sched-cal-shell">
      <header class="sched-cal-head">
        <h3>{{ title }}</h3>
        <button type="button" class="sched-cal-close" @click="visible = false">关闭</button>
      </header>

      <div v-if="members.length > 1" class="sched-cal-member-row">
        <van-field
          :model-value="activeMemberLabel"
          is-link
          readonly
          label="员工"
          placeholder="选择员工查看排班"
          class="sched-cal-member-field"
          @click="showMemberSheet = true"
        />
      </div>

      <div v-if="showLegend" class="sched-cal-legend" aria-label="班次图例">
        <span
          v-for="item in SCHEDULE_TAG_LEGEND"
          :key="item.code"
          class="sched-cal-legend-chip"
          :class="`sched-cal-legend-chip--${item.kind}`"
        >
          {{ item.label }}
        </span>
      </div>

      <div v-if="previewYmd" class="sched-cal-preview">
        <span class="sched-cal-preview-date">{{ previewYmd }}</span>
        <p class="sched-cal-preview-text">{{ previewSummary }}</p>
      </div>

      <van-calendar
        :key="calendarKey"
        switch-mode="month"
        :poppable="false"
        :show-confirm="true"
        confirm-text="选择此日"
        :min-date="minDate"
        :max-date="maxDate"
        :default-date="calendarAnchor"
        :formatter="scheduleFormatter"
        @select="onCalendarSelect"
        @confirm="onConfirm"
        @panel-change="onPanelChange"
      />
      <p class="sched-cal-hint">切换员工后月历显示对应排班；点击日期可预览，确认后返回</p>
    </div>

    <van-action-sheet
      v-model:show="showMemberSheet"
      :actions="memberActions"
      cancel-text="取消"
      description="选择要查看排班的员工"
      close-on-click-action
      @select="onPickMemberAction"
    />
  </van-popup>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { getScheduleDay, getScheduleMonth } from '../api'
import { toLocalYMD } from '../lib/fortuneUtils'
import {
  SCHEDULE_TAG_LEGEND,
  formatScheduleCalendarBottom,
  formatScheduleDaySummary,
  scheduleDayCssClass,
} from '../lib/scheduleCalendarLabel'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  userId: { type: Number, required: true },
  initialDate: { type: String, default: '' },
  title: { type: String, default: '选择日期' },
  popupHeight: { type: String, default: '88%' },
  showLegend: { type: Boolean, default: false },
  members: {
    type: Array,
    default: () => [],
  },
  memberIndex: { type: Number, default: 0 },
})

const emit = defineEmits(['update:modelValue', 'update:memberIndex', 'confirm'])

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const activeMemberIndex = computed({
  get: () => props.memberIndex,
  set: (value) => emit('update:memberIndex', value),
})

const activeUserId = computed(() => {
  const member = props.members[activeMemberIndex.value]
  if (member?.userId) return Number(member.userId)
  return props.userId
})

const activeMemberLabel = computed(() => {
  const member = props.members[activeMemberIndex.value]
  if (!member) return '选择员工'
  const name = member.userName || `员工 ${member.userId}`
  const account = member.account ? `（${member.account}）` : ''
  return `${name}${account}`
})

const memberActions = computed(() =>
  props.members.map((member, index) => ({
    name: member.userName || `员工 ${member.userId}`,
    subname: member.account || undefined,
    index,
  })),
)

const minDate = new Date(new Date().getFullYear() - 1, 0, 1)
const maxDate = new Date(new Date().getFullYear() + 1, 11, 31)

const scheduleMonthDays = ref({})
const calendarKey = ref(0)
const panelAnchor = ref(null)
const showMemberSheet = ref(false)
const previewYmd = ref('')
const previewSchedule = ref(null)

const anchorYmd = computed(() => props.initialDate || toLocalYMD(new Date()))
const calendarAnchor = computed(() => new Date(`${anchorYmd.value}T12:00:00`))

const previewSummary = computed(() => formatScheduleDaySummary(previewSchedule.value))

const loadScheduleMonth = async (anchorDate) => {
  const d = anchorDate instanceof Date ? anchorDate : calendarAnchor.value
  const month = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
  try {
    const { data } = await getScheduleMonth(activeUserId.value, month)
    scheduleMonthDays.value = data.days || {}
  } catch {
    scheduleMonthDays.value = {}
  }
}

const loadDayPreview = async (ymd) => {
  previewYmd.value = ymd
  try {
    const { data } = await getScheduleDay(activeUserId.value, ymd)
    previewSchedule.value = data
  } catch {
    previewSchedule.value = null
  }
}

const scheduleFormatter = (day) => {
  const ymd = toLocalYMD(day.date)
  const info = scheduleMonthDays.value[ymd]
  const bottomInfo = formatScheduleCalendarBottom(info)
  if (bottomInfo) {
    return { ...day, bottomInfo, className: scheduleDayCssClass(info) }
  }
  return day
}

const onPanelChange = ({ date }) => {
  if (!date) return
  panelAnchor.value = date
  void loadScheduleMonth(date)
}

const pickMember = async (index) => {
  activeMemberIndex.value = index
  calendarKey.value += 1
  await loadScheduleMonth(panelAnchor.value || calendarAnchor.value)
  await loadDayPreview(previewYmd.value || anchorYmd.value)
}

const onPickMemberAction = (action) => {
  if (action && typeof action.index === 'number') {
    void pickMember(action.index)
  }
}

const onCalendarSelect = (value) => {
  const ymd = toLocalYMD(value instanceof Date ? value : new Date(value))
  void loadDayPreview(ymd)
}

const onConfirm = (value) => {
  const d = value instanceof Date ? value : new Date(value)
  emit('confirm', toLocalYMD(d))
  visible.value = false
}

watch(
  () => props.modelValue,
  async (open) => {
    if (!open) return
    calendarKey.value += 1
    panelAnchor.value = calendarAnchor.value
    previewYmd.value = anchorYmd.value
    await loadScheduleMonth(calendarAnchor.value)
    await loadDayPreview(anchorYmd.value)
  },
)

watch(activeUserId, async () => {
  if (!props.modelValue) return
  await loadScheduleMonth(panelAnchor.value || calendarAnchor.value)
  if (previewYmd.value) await loadDayPreview(previewYmd.value)
})
</script>

<style scoped>
.sched-cal-shell {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 12px 16px;
  background: #f8fafc;
}
.sched-cal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.sched-cal-head h3 {
  margin: 0;
  font-size: 17px;
  color: #102a5c;
}
.sched-cal-close {
  border: 1px solid #d8e0f5;
  background: #fff;
  color: #2c5ee8;
  border-radius: 999px;
  padding: 6px 14px;
  font-size: 13px;
}
.sched-cal-member-row {
  margin-bottom: 4px;
}
.sched-cal-member-field {
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #d8e0f5;
}
.sched-cal-member-field :deep(.van-field__label) {
  color: #64748b;
  width: 3em;
}
.sched-cal-member-field :deep(.van-field__control) {
  color: #102a5c;
  font-weight: 600;
}
.sched-cal-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}
.sched-cal-legend-chip {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
  background: #fff;
  color: #475569;
}
.sched-cal-legend-chip--leave {
  border-color: #bbf7d0;
  background: #f0fdf4;
  color: #166534;
}
.sched-cal-legend-chip--work {
  border-color: #bfdbfe;
  background: #eff6ff;
  color: #1d4ed8;
}
.sched-cal-preview {
  margin-bottom: 8px;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid #dbe5ff;
  background: #fff;
}
.sched-cal-preview-date {
  display: block;
  font-size: 12px;
  color: #64748b;
  margin-bottom: 4px;
}
.sched-cal-preview-text {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #102a5c;
  line-height: 1.4;
}
.sched-cal-hint {
  margin: 8px 0 0;
  text-align: center;
  font-size: 12px;
  color: #64748b;
}
:deep(.schedule-cal-day--shift) {
  color: #1d4ed8;
  font-size: 10px;
  font-weight: 700;
  line-height: 1.1;
}
:deep(.schedule-cal-day--leave) {
  color: #166534;
  font-size: 10px;
  font-weight: 700;
}
:deep(.schedule-cal-day--standby) {
  color: #7c3aed;
  font-size: 9px;
  font-weight: 700;
}
:deep(.van-calendar__body) {
  flex: 1;
  overflow: hidden;
}
</style>
