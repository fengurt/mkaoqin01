<template>
  <div>
    <van-button
      class="voice-btn"
      :type="isRecording ? 'danger' : 'primary'"
      block
      round
      @touchstart.prevent="startRecording"
      @touchend.prevent="stopRecording"
      @mousedown.prevent="startRecording"
      @mouseup.prevent="stopRecording"
    >
      {{ isRecording ? '正在录音...' : '按住说话' }}
    </van-button>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['recorded'])
const isRecording = ref(false)
let mediaRecorder = null
let chunks = []

const startRecording = async () => {
  if (isRecording.value) return
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  mediaRecorder = new MediaRecorder(stream)
  chunks = []
  mediaRecorder.ondataavailable = (event) => chunks.push(event.data)
  mediaRecorder.onstop = () => {
    const audioBlob = new Blob(chunks, { type: 'audio/wav' })
    emit('recorded', audioBlob)
    stream.getTracks().forEach((track) => track.stop())
  }
  mediaRecorder.start()
  isRecording.value = true
}

const stopRecording = () => {
  if (!isRecording.value || !mediaRecorder) return
  mediaRecorder.stop()
  isRecording.value = false
}
</script>

<style scoped>
.voice-btn {
  height: 50px;
  font-size: 16px;
}
</style>
