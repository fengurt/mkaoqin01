<template>
  <div class="screen-page">
    <header class="topbar">
      <button class="icon-btn" type="button" @click="$router.push('/me/accounts')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <h1>今日好运 · {{ displayName }}</h1>
      <span class="material-symbols-outlined">auto_awesome</span>
    </header>

    <main class="content">
      <section
        class="drop-zone"
        :class="{ 'drop-zone--active': dragOver }"
        @dragover.prevent="dragOver = true"
        @dragleave.prevent="dragOver = false"
        @drop.prevent="onDropFile"
      >
        <p class="drop-title">选中日期：{{ selectedDate }}</p>
        <div v-if="previewImageSrc" class="preview-wrap">
          <img :src="previewImageSrc" alt="预览" class="preview-img" />
          <p class="preview-caption">{{ previewCaption || '（无话术）' }}</p>
          <div class="preview-actions">
            <van-button size="small" type="primary" plain @click="openSyncDialog">同步到其他员工</van-button>
            <van-button size="small" type="primary" @click="triggerFilePick">更换图片</van-button>
          </div>
        </div>
        <div v-else class="drop-empty">
          <span class="material-symbols-outlined">upload</span>
          <p>拖拽图片到此处，或点击上传</p>
          <van-button size="small" type="primary" @click="triggerFilePick">选择图片</van-button>
        </div>
        <van-field v-model="captionDraft" label="话术" placeholder="可选，显示在海报下方" />
        <van-button v-if="previewImageSrc" size="small" plain type="primary" block @click="saveCaptionOnly">
          保存话术（不换图）
        </van-button>
        <input ref="fileInputRef" type="file" accept="image/*" class="hidden-input" @change="onFileInput" />
      </section>

      <section class="cal-card">
        <van-calendar
          :poppable="false"
          :show-confirm="false"
          :min-date="minDate"
          :max-date="maxDate"
          :default-date="calendarAnchor"
          :formatter="calendarFormatter"
          @select="onSelectDay"
          @panel-change="onPanelChange"
        />
      </section>

      <section v-if="monthItems.length" class="thumb-card">
        <h3>本月已配置</h3>
        <div class="thumb-grid">
          <button
            v-for="item in monthItems"
            :key="item.date"
            type="button"
            class="thumb-cell"
            :class="{ active: item.date === selectedDate }"
            @click="selectFromThumb(item)"
          >
            <img v-if="item.imageUrl" :src="resolveFortuneImageSrc(item.imageUrl)" alt="" />
            <span>{{ item.date.slice(8) }}日</span>
          </button>
        </div>
      </section>
    </main>

    <van-popup v-model:show="showSync" position="bottom" round :style="{ height: '70%' }">
      <div class="sync-popup">
        <h3>同步到其他员工</h3>
        <p class="sync-sub">将 {{ displayName }} · {{ selectedDate }} 的海报复制到所选员工同一日期</p>
        <van-field v-model="syncKeyword" placeholder="搜索员工" clearable />
        <van-checkbox-group v-model="syncTargetIds">
          <van-cell-group inset>
            <van-cell
              v-for="u in filteredSyncUsers"
              :key="u.id"
              :title="u.displayName"
              :label="u.account"
              clickable
              @click="toggleSyncUser(u.id)"
            >
              <template #right-icon>
                <van-checkbox :name="u.id" @click.stop />
              </template>
            </van-cell>
          </van-cell-group>
        </van-checkbox-group>
        <van-button type="primary" block :loading="syncBusy" @click="submitSync">确认同步</van-button>
      </div>
    </van-popup>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { showFailToast, showSuccessToast } from 'vant'
import {
  assignAdminFortune,
  getAdminFortuneMonth,
  getAuthUsers,
  syncAdminFortune,
  uploadAdminFortune,
} from '../../api'
import { buildFortuneMonthMap, monthRange, resolveFortuneImageSrc, toLocalYMD } from '../../lib/fortuneUtils'

const route = useRoute()
const employeeUserId = computed(() => Number(route.params.userId) || 0)
const displayName = computed(() => String(route.query.name || '员工'))

const selectedDate = ref(toLocalYMD(new Date()))
const monthMap = ref({})
const captionDraft = ref('')
const previewImageSrc = ref('')
const previewCaption = ref('')
const dragOver = ref(false)
const fileInputRef = ref(null)
const showSync = ref(false)
const syncKeyword = ref('')
const syncTargetIds = ref([])
const syncBusy = ref(false)
const allUsers = ref([])

const minDate = new Date(new Date().getFullYear() - 1, 0, 1)
const maxDate = new Date(new Date().getFullYear() + 1, 11, 31)
const calendarAnchor = computed(() => new Date(`${selectedDate.value}T12:00:00`))

const monthItems = computed(() =>
  Object.values(monthMap.value).sort((a, b) => String(a.date).localeCompare(String(b.date))),
)

const filteredSyncUsers = computed(() => {
  const key = syncKeyword.value.trim().toLowerCase()
  return allUsers.value.filter((u) => {
    if (u.id === employeeUserId.value) return false
    if (u.role === 'admin') return false
    if (!key) return true
    return (
      String(u.displayName || '').toLowerCase().includes(key) ||
      String(u.account || '').toLowerCase().includes(key)
    )
  })
})

const loadMonth = async () => {
  const { from, to } = monthRange(selectedDate.value)
  try {
    const { data } = await getAdminFortuneMonth(employeeUserId.value, from, to)
    monthMap.value = buildFortuneMonthMap(data.items)
    applySelectedFromMap()
  } catch {
    monthMap.value = {}
    previewImageSrc.value = ''
    previewCaption.value = ''
  }
}

const applySelectedFromMap = () => {
  const row = monthMap.value[selectedDate.value]
  if (row) {
    previewImageSrc.value = resolveFortuneImageSrc(row.imageUrl)
    previewCaption.value = row.caption || ''
    captionDraft.value = row.caption || ''
  } else {
    previewImageSrc.value = ''
    previewCaption.value = ''
    captionDraft.value = ''
  }
}

const calendarFormatter = (day) => {
  const ymd = toLocalYMD(day.date)
  if (monthMap.value[ymd]?.imageUrl) {
    return { ...day, bottomInfo: '签', className: 'fortune-day--assigned' }
  }
  return day
}

const onSelectDay = async (value) => {
  selectedDate.value = toLocalYMD(value)
  applySelectedFromMap()
}

const onPanelChange = async ({ date }) => {
  if (date) await loadMonth(date)
}

const saveCaptionOnly = async () => {
  const row = monthMap.value[selectedDate.value]
  const imageUrl = row?.imageUrl
  if (!imageUrl) {
    showFailToast('请先上传图片')
    return
  }
  try {
    await assignAdminFortune({
      userId: employeeUserId.value,
      date: selectedDate.value,
      imageUrl,
      caption: captionDraft.value.trim(),
    })
    showSuccessToast('话术已保存')
    await loadMonth()
  } catch (error) {
    showFailToast(error?.response?.data?.error || '保存失败')
  }
}

const selectFromThumb = (item) => {
  selectedDate.value = item.date
  previewImageSrc.value = resolveFortuneImageSrc(item.imageUrl)
  previewCaption.value = item.caption || ''
  captionDraft.value = item.caption || ''
}

const triggerFilePick = () => fileInputRef.value?.click()

const uploadFile = async (file) => {
  if (!file || !employeeUserId.value) return
  const form = new FormData()
  form.append('image', file)
  form.append('userId', String(employeeUserId.value))
  form.append('date', selectedDate.value)
  if (captionDraft.value.trim()) form.append('caption', captionDraft.value.trim())
  try {
    const { data } = await uploadAdminFortune(form)
    previewImageSrc.value = resolveFortuneImageSrc(data.imageUrl)
    previewCaption.value = data.caption || ''
    showSuccessToast('已上传并保存')
    await loadMonth()
  } catch (error) {
    showFailToast(error?.response?.data?.error || '上传失败')
  }
}

const onFileInput = async (event) => {
  const file = event.target.files?.[0]
  event.target.value = ''
  await uploadFile(file)
}

const onDropFile = async (event) => {
  dragOver.value = false
  const file = event.dataTransfer?.files?.[0]
  await uploadFile(file)
}

const loadUsers = async () => {
  try {
    const { data } = await getAuthUsers()
    const raw = data.items ?? data.users ?? []
    allUsers.value = Array.isArray(raw) ? raw : []
  } catch {
    allUsers.value = []
  }
}

const openSyncDialog = () => {
  if (!previewImageSrc.value) {
    showFailToast('请先为该日期上传海报')
    return
  }
  syncTargetIds.value = []
  syncKeyword.value = ''
  showSync.value = true
}

const toggleSyncUser = (id) => {
  const set = new Set(syncTargetIds.value)
  if (set.has(id)) set.delete(id)
  else set.add(id)
  syncTargetIds.value = [...set]
}

const submitSync = async () => {
  if (!syncTargetIds.value.length) {
    showFailToast('请至少选择一名员工')
    return
  }
  syncBusy.value = true
  try {
    const { data } = await syncAdminFortune({
      sourceUserId: employeeUserId.value,
      date: selectedDate.value,
      targetUserIds: syncTargetIds.value.map(Number),
    })
    showSuccessToast(`已同步 ${data.synced || 0} 人`)
    showSync.value = false
  } catch (error) {
    showFailToast(error?.response?.data?.error || '同步失败')
  } finally {
    syncBusy.value = false
  }
}

watch(selectedDate, applySelectedFromMap)

onMounted(async () => {
  await loadUsers()
  await loadMonth()
})
</script>

<style scoped>
.screen-page { min-height: 100vh; background: #fff9f0; }
.topbar {
  position: sticky; top: 0; z-index: 20; height: 64px;
  border-bottom: 1px solid #fed7aa; background: #fff;
  display: flex; align-items: center; justify-content: space-between; padding: 0 12px;
}
.topbar h1 { margin: 0; font-size: 16px; color: #7c2d12; max-width: 70%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.icon-btn {
  width: 36px; height: 36px; border: 1px solid #fed7aa; border-radius: 999px;
  background: #fff; color: #c2410c; display: flex; align-items: center; justify-content: center;
}
.content { padding: 12px; display: flex; flex-direction: column; gap: 12px; }
.drop-zone {
  border: 2px dashed #fdba74; border-radius: 14px; padding: 14px; background: #fff;
  transition: border-color 0.15s, background 0.15s;
}
.drop-zone--active { border-color: #ea580c; background: #fff7ed; }
.drop-title { margin: 0 0 10px; font-size: 13px; color: #9a3412; }
.drop-empty { text-align: center; color: #a8a29e; padding: 20px 0; }
.drop-empty span { font-size: 36px; color: #fdba74; }
.preview-wrap { display: flex; flex-direction: column; gap: 8px; }
.preview-img { width: 100%; max-height: 220px; object-fit: contain; border-radius: 10px; }
.preview-caption { margin: 0; font-size: 14px; color: #44403c; }
.preview-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.hidden-input { display: none; }
.cal-card { background: #fff; border-radius: 14px; overflow: hidden; border: 1px solid #fed7aa; }
.thumb-card { background: #fff; border-radius: 14px; padding: 12px; border: 1px solid #fed7aa; }
.thumb-card h3 { margin: 0 0 10px; font-size: 14px; color: #7c2d12; }
.thumb-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.thumb-cell {
  border: 1px solid #e7e5e4; border-radius: 8px; padding: 4px; background: #fafaf9;
  display: flex; flex-direction: column; align-items: center; gap: 4px; font-size: 11px;
}
.thumb-cell.active { border-color: #ea580c; box-shadow: 0 0 0 2px rgba(234, 88, 12, 0.2); }
.thumb-cell img { width: 100%; height: 52px; object-fit: cover; border-radius: 4px; }
.sync-popup { padding: 16px; display: flex; flex-direction: column; gap: 10px; max-height: 100%; overflow: auto; }
.sync-popup h3 { margin: 0; }
.sync-sub { margin: 0; font-size: 12px; color: #78716c; }
:deep(.fortune-day--assigned) { color: #c2410c; font-weight: 700; }
</style>
