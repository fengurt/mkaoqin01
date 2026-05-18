<template>
  <van-popup v-model:show="visible" position="bottom" round :style="{ height: '78%' }">
    <div class="guide-shell">
      <header class="guide-head">
        <h3>班次 JSON · Agent 指南</h3>
        <button type="button" class="guide-close" @click="visible = false">关闭</button>
      </header>
      <p class="guide-path">
        仓库路径：<code>importdata/SCHEDULE_GRID_AGENT_GUIDE.md</code>
      </p>
      <div v-if="loading" class="guide-loading">加载中…</div>
      <pre v-else class="guide-body">{{ guideText }}</pre>
    </div>
  </van-popup>
</template>

<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const guideText = ref('')
const loading = ref(false)

const loadGuide = async () => {
  loading.value = true
  try {
    const res = await fetch('/importdata/SCHEDULE_GRID_AGENT_GUIDE.md')
    guideText.value = res.ok ? await res.text() : '指南文件未找到，请查看仓库 importdata/SCHEDULE_GRID_AGENT_GUIDE.md'
  } catch {
    guideText.value = '无法加载指南，请查看仓库 importdata/SCHEDULE_GRID_AGENT_GUIDE.md'
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (open) => {
    if (open && !guideText.value) loadGuide()
  },
)
</script>

<style scoped>
.guide-shell {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 14px 16px;
  background: #f8fafc;
}
.guide-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.guide-head h3 {
  margin: 0;
  font-size: 16px;
  color: #102a5c;
}
.guide-close {
  border: 1px solid #d8e0f5;
  background: #fff;
  color: #2c5ee8;
  border-radius: 999px;
  padding: 5px 12px;
  font-size: 12px;
}
.guide-path {
  margin: 0 0 8px;
  font-size: 11px;
  color: #64748b;
}
.guide-path code {
  background: #e2e8f0;
  padding: 1px 4px;
  border-radius: 4px;
  font-size: 10px;
}
.guide-loading {
  padding: 24px;
  text-align: center;
  color: #64748b;
  font-size: 13px;
}
.guide-body {
  flex: 1;
  margin: 0;
  overflow: auto;
  padding: 10px;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
  background: #fff;
  font-size: 11px;
  line-height: 1.45;
  color: #334155;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
