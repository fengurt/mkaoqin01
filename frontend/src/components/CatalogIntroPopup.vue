<template>
  <van-popup
    :show="modelValue"
    position="bottom"
    round
    :style="{ height: '72%' }"
    @update:show="$emit('update:modelValue', $event)"
  >
    <div class="intro-sheet">
      <div class="intro-sheet-head">
        <h3>{{ title }}</h3>
        <button type="button" class="intro-close" @click="$emit('update:modelValue', false)">关闭</button>
      </div>
      <div class="intro-sheet-body">
        <pre class="intro-detail">{{ detail }}</pre>
      </div>
    </div>
  </van-popup>
</template>

<script setup>
defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  detail: { type: String, default: '' },
})

defineEmits(['update:modelValue'])
</script>

<style scoped>
.intro-sheet {
  padding: 14px 14px 22px;
  height: 100%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}
.intro-sheet-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 10px;
  flex-shrink: 0;
}
.intro-sheet-head h3 {
  margin: 0;
  font-size: 17px;
  color: #0f172a;
  flex: 1;
  min-width: 0;
}
.intro-close {
  border: 0;
  background: transparent;
  color: #3156cb;
  font-size: 14px;
  flex-shrink: 0;
}
.intro-sheet-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.intro-detail {
  margin: 0;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
  color: #334155;
}
</style>
