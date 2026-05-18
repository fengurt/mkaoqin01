<template>
  <van-popup v-model:show="visible" position="bottom" round :style="{ height: '88%' }">
    <div class="fortune-cal-shell">
      <header class="fortune-cal-head">
        <h3>{{ title }}</h3>
        <button type="button" class="fortune-cal-close" @click="visible = false">关闭</button>
      </header>
      <van-calendar
        :poppable="false"
        :show-confirm="false"
        :min-date="minDate"
        :max-date="maxDate"
        :default-date="calendarAnchor"
        :formatter="calendarFormatter"
        @select="onSelectDay"
      />
      <p class="fortune-cal-hint">点击日期查看当日海报或话术</p>
    </div>
  </van-popup>

  <van-popup v-model:show="showPoster" round :style="{ width: '92%', maxWidth: '420px' }">
    <div class="fortune-poster">
      <img
        v-if="posterImageSrc"
        :src="posterImageSrc"
        alt="今日好运"
        class="fortune-poster-img"
        @error="onPosterImageError"
      />
      <p class="fortune-poster-caption">{{ posterCaption }}</p>
      <p v-if="posterSource === 'pool'" class="fortune-poster-meta">系统随机签</p>
      <p v-else class="fortune-poster-meta">专属定制</p>
    </div>
  </van-popup>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { showFailToast } from 'vant'
import { getFortuneDay, getFortuneMonth } from '../api'
import { buildFortuneMonthMap, monthRange, resolveFortuneImageSrc, toLocalYMD } from '../lib/fortuneUtils'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  userId: { type: Number, required: true },
  title: { type: String, default: '今日好运' },
  initialDate: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const minDate = new Date(new Date().getFullYear() - 1, 0, 1)
const maxDate = new Date(new Date().getFullYear() + 1, 11, 31)

const monthMap = ref({})
const showPoster = ref(false)
const posterImageSrc = ref('')
const posterCaption = ref('')
const posterSource = ref('pool')
const selectedYmd = ref('')

const calendarAnchor = computed(() => {
  const raw = props.initialDate || toLocalYMD(new Date())
  return new Date(`${raw}T12:00:00`)
})

const loadMonth = async (anchor) => {
  const { from, to } = monthRange(anchor)
  try {
    const { data } = await getFortuneMonth(props.userId, from, to)
    monthMap.value = buildFortuneMonthMap(data.items)
  } catch {
    monthMap.value = {}
  }
}

watch(
  () => props.modelValue,
  async (open) => {
    if (!open) return
    selectedYmd.value = props.initialDate || toLocalYMD(new Date())
    await loadMonth(calendarAnchor.value)
    await openDayFortune(selectedYmd.value)
  },
)

const calendarFormatter = (day) => {
  const ymd = toLocalYMD(day.date)
  const row = monthMap.value[ymd]
  if (row?.imageUrl) {
    return { ...day, bottomInfo: '签', className: 'fortune-day--assigned' }
  }
  return { ...day, bottomInfo: '·', className: 'fortune-day--pool' }
}

const openDayFortune = async (ymd) => {
  try {
    const { data } = await getFortuneDay(props.userId, ymd)
    posterImageSrc.value = resolveFortuneImageSrc(data.imageUrl)
    posterCaption.value = data.caption || ''
    posterSource.value = data.source || 'pool'
    showPoster.value = true
  } catch {
    showFailToast('加载当日好运失败')
  }
}

const onSelectDay = async (value) => {
  const ymd = toLocalYMD(value)
  selectedYmd.value = ymd
  await loadMonth(value)
  await openDayFortune(ymd)
}

const onPosterImageError = () => {
  posterImageSrc.value = ''
}
</script>

<style scoped>
.fortune-cal-shell {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 12px 20px;
  background: linear-gradient(180deg, #fff9f0 0%, #fff 40%);
}
.fortune-cal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.fortune-cal-head h3 {
  margin: 0;
  font-size: 18px;
  color: #7c2d12;
}
.fortune-cal-close {
  border: 1px solid #fed7aa;
  background: #fff;
  color: #c2410c;
  border-radius: 999px;
  padding: 6px 14px;
  font-size: 13px;
}
.fortune-cal-hint {
  margin: 8px 0 0;
  text-align: center;
  font-size: 12px;
  color: #9a3412;
}
:deep(.fortune-day--assigned) {
  color: #c2410c;
  font-weight: 700;
}
:deep(.fortune-day--pool) {
  color: #d6d3d1;
}
.fortune-poster {
  padding: 16px;
  text-align: center;
}
.fortune-poster-img {
  width: 100%;
  border-radius: 12px;
  display: block;
}
.fortune-poster-caption {
  margin: 12px 0 4px;
  font-size: 15px;
  line-height: 1.5;
  color: #292524;
}
.fortune-poster-meta {
  margin: 0;
  font-size: 12px;
  color: #a8a29e;
}
</style>
