<template>
  <van-popup :show="modelValue" position="bottom" round :style="{ height: '72%' }" @update:show="$emit('update:modelValue', $event)">
    <div class="dsp">
      <div class="dsp-head">
        <h3 class="dsp-title">当日排班类型</h3>
        <button type="button" class="dsp-close" @click="$emit('update:modelValue', false)">关闭</button>
      </div>
      <p class="dsp-meta-line">{{ dateStr }} · 仅可选一项 · 与时间轴等联动</p>
      <van-loading v-if="optionsLoading" class="dsp-loading" />
      <div v-else-if="optionsErrorMessage" class="dsp-empty">
        <p class="dsp-empty-text">{{ optionsErrorMessage }}</p>
        <van-button type="primary" size="small" round plain hairline @click="retryLoadOptions">重试</van-button>
      </div>
      <div v-else-if="!workShifts.length && !leaveTypes.length" class="dsp-empty">
        <p class="dsp-empty-text">暂无班次与假别数据，请在后台维护常规班班次与假期类型后重试。</p>
        <van-button type="primary" size="small" round plain hairline @click="retryLoadOptions">重新加载</van-button>
      </div>
      <div v-else class="dsp-body">
        <div class="dsp-scroll">
          <section class="dsp-section" aria-labelledby="dsp-head-work">
            <h4 id="dsp-head-work" class="dsp-section-head">常规班</h4>
            <div class="dsp-opt-stack">
              <button
                v-for="s in workShifts"
                :key="s.code"
                type="button"
                class="dsp-opt-line"
                :class="{ 'dsp-opt-line--active': isWorkRowActive(s.code) }"
                :disabled="persisting"
                @click="persist('work', s.code)"
              >
                <span class="dsp-opt-text">{{ workOneLine(s) }}</span>
                <span class="dsp-opt-code" aria-hidden="true">{{ s.code }}</span>
                <span class="dsp-opt-icon-slot" aria-hidden="true">
                  <span v-if="isWorkRowActive(s.code)" class="material-symbols-outlined dsp-opt-check">check_circle</span>
                </span>
              </button>
            </div>
          </section>

          <section class="dsp-section" aria-labelledby="dsp-head-leave">
            <h4 id="dsp-head-leave" class="dsp-section-head">休假</h4>
            <div class="dsp-opt-stack">
              <button
                v-for="l in leaveTypes"
                :key="l.code"
                type="button"
                class="dsp-opt-line"
                :class="{ 'dsp-opt-line--active': isLeaveRowActive(l.code) }"
                :disabled="persisting"
                @click="persist('leave', l.code)"
              >
                <span class="dsp-opt-text">{{ leaveOneLine(l) }}</span>
                <span class="dsp-opt-code dsp-opt-code--muted" aria-hidden="true">{{ leaveShowCodeChip(l) ? l.code : '' }}</span>
                <span class="dsp-opt-icon-slot" aria-hidden="true">
                  <span v-if="isLeaveRowActive(l.code)" class="material-symbols-outlined dsp-opt-check">check_circle</span>
                </span>
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>
  </van-popup>
</template>

<script setup>
import { ref, watch } from 'vue'
import { showFailToast, showSuccessToast } from 'vant'
import { getScheduleDayOptions, setScheduleDay } from '../api'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  dateStr: { type: String, required: true },
  userId: { type: Number, default: 1 },
  resolvedSchedule: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'applied'])

const optionsLoading = ref(false)
const workShifts = ref([])
const leaveTypes = ref([])
const optionsErrorMessage = ref('')
const optionsLoadFailed = ref(false)
const persisting = ref(false)

/** Mirrors server state for highlighting; updated when popup opens and after save */
const activeSchedule = ref(null)

/** 左侧已有分组标题，行内不再重复「常规班-」「休假-」 */
const workOneLine = (s) => `${s.name} · ${s.startTime}–${s.endTime}`

/** 假期一行展示：代码右侧另有小号代码列，主文案只保留可读名称 */
const leaveOneLine = (l) => {
  const nameZh = (l.description || '').trim() || (l.fullName || '').trim()
  return nameZh || l.code
}

const leaveShowCodeChip = (l) => leaveOneLine(l) !== l.code

const isWorkRowActive = (code) => activeSchedule.value?.mode === 'work' && activeSchedule.value?.code === code

const isLeaveRowActive = (code) => activeSchedule.value?.mode === 'leave' && activeSchedule.value?.code === code

const syncActiveFromProps = () => {
  const rs = props.resolvedSchedule
  if (!rs || !rs.mode || !rs.code) {
    activeSchedule.value = null
    return
  }
  activeSchedule.value = {
    mode: rs.mode,
    code: rs.code,
    pillText: rs.pillText,
  }
}

const loadOptionsOnce = async (force = false) => {
  if (!force && workShifts.value.length && !optionsLoadFailed.value) return
  optionsLoading.value = true
  optionsLoadFailed.value = false
  optionsErrorMessage.value = ''
  try {
    const { data } = await getScheduleDayOptions()
    workShifts.value = Array.isArray(data?.workShifts) ? data.workShifts : []
    leaveTypes.value = Array.isArray(data?.leaveTypes) ? data.leaveTypes : []
  } catch (error) {
    optionsLoadFailed.value = true
    const status = error?.response?.status
    const hint =
      status === 401
        ? '登录已过期或未登录（选项列表需联网加载）'
        : status === 404 || error?.code === 'ERR_NETWORK'
          ? '无法连接考勤服务，请确认网关已启动'
          : '排班选项加载失败'
    optionsErrorMessage.value = hint
    showFailToast(hint)
  } finally {
    optionsLoading.value = false
  }
}

const retryLoadOptions = () => loadOptionsOnce(true)

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      syncActiveFromProps()
      loadOptionsOnce()
    }
  },
)

watch(
  () => props.resolvedSchedule,
  () => {
    if (props.modelValue) syncActiveFromProps()
  },
  { deep: true },
)

const persistErrorToast = (error) => {
  const code = error?.response?.data?.error
  const msg =
    code === 'schedule_db_pending'
      ? '排班表未就绪，请重启考勤服务后再试'
      : code === 'schedule_user_unknown'
        ? '当前用户不存在于系统用户表，请用账号登录或联系管理员'
        : code === 'unknown shift code' || code === 'unknown leave code'
          ? '所选班次已失效，请重新加载排班选项'
          : '保存失败'
  showFailToast(msg)
}

const persist = async (mode, code) => {
  if (persisting.value) return
  persisting.value = true
  try {
    const { data } = await setScheduleDay({
      userId: props.userId || 1,
      date: props.dateStr,
      mode,
      code,
    })
    activeSchedule.value = {
      mode: data?.mode || mode,
      code: data?.code || code,
      pillText: data?.pillText,
    }
    showSuccessToast('已更新当日排班')
    emit('applied', data)
  } catch (error) {
    persistErrorToast(error)
  } finally {
    persisting.value = false
  }
}
</script>

<style scoped>
.dsp {
  padding: 12px 0 16px;
  display: flex;
  flex-direction: column;
  max-height: 100%;
  min-height: 0;
}
.dsp-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 14px 6px;
  flex-shrink: 0;
}
.dsp-title {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  color: var(--brand-title, #0f172a);
}
.dsp-close {
  border: 0;
  background: transparent;
  color: var(--brand-primary-mid, #2563eb);
  font-size: 14px;
}
.dsp-meta-line {
  margin: 0 14px 8px;
  font-size: 11px;
  color: var(--brand-subtext, #64748b);
  line-height: 1.25;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 0;
}
.dsp-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0 10px;
}
.dsp-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  padding-bottom: 4px;
}
.dsp-section {
  margin-bottom: 14px;
}
.dsp-section:last-child {
  margin-bottom: 0;
}
.dsp-section-head {
  margin: 0 2px 5px;
  padding: 0;
  font-size: 12px;
  font-weight: 800;
  color: var(--brand-subtext, #64748b);
  letter-spacing: 0.03em;
}
.dsp-opt-stack {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.dsp-opt-line {
  width: 100%;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  column-gap: 8px;
  text-align: left;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 5px 8px;
  min-height: 32px;
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  color: #0f172a;
  transition:
    border-color 0.12s ease,
    background 0.12s ease;
  -webkit-tap-highlight-color: transparent;
}
.dsp-opt-line:disabled {
  opacity: 0.65;
  cursor: wait;
}
.dsp-opt-line:active:not(:disabled) {
  filter: brightness(0.98);
}
.dsp-opt-line--active {
  border-color: #2563eb;
  background: #eff6ff;
}
.dsp-opt-text {
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dsp-opt-code {
  flex-shrink: 0;
  justify-self: end;
  min-width: 56px;
  max-width: 78px;
  text-align: right;
  font-size: 10px;
  font-weight: 600;
  font-family: ui-monospace, monospace;
  color: #64748b;
  opacity: 0.92;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dsp-opt-code--muted {
  opacity: 0.75;
}
.dsp-opt-icon-slot {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.dsp-opt-check {
  font-size: 18px;
  width: 18px;
  height: 18px;
  color: #2563eb;
}
.dsp-loading {
  margin: 24px auto;
  display: flex;
  justify-content: center;
}
.dsp-empty {
  padding: 28px 20px 16px;
  text-align: center;
}
.dsp-empty-text {
  margin: 0 0 14px;
  font-size: 13px;
  line-height: 1.55;
  color: var(--brand-subtext, #64748b);
}
</style>
