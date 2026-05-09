<template>
  <div class="screen-page">
    <header class="topbar">
      <button class="icon-btn" type="button" @click="$router.push('/me')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <h1>班次与假期</h1>
      <span class="material-symbols-outlined tune-ico">schedule</span>
    </header>

    <main class="content">
      <p class="lead">
        维护<strong>常规班</strong>（<code>shift_types</code>）与<strong>假期类型</strong>（<code>activity_types</code>）。保存后员工在「行程 / 申报」里下拉即可使用；时间轴药丸显示
        <code>常规班-班次名 时段</code>。
      </p>

      <van-tabs v-model:active="tabActive" shrink sticky animated>
        <van-tab title="常规班班次">
          <section class="card tab-card">
            <div class="head-row">
              <h3>班次列表</h3>
              <van-button size="small" type="primary" round @click="openShiftEditor(null)">新增</van-button>
            </div>
            <p v-if="shiftError" class="error-block">{{ shiftError }}</p>
            <p v-else-if="!shifts.length" class="empty">暂无数据，请新增或检查数据库 <code>shift_types</code>。</p>
            <div v-else class="list-wrap">
              <div v-for="row in shifts" :key="row.code" class="catalog-row">
                <div class="data-main">
                  <div class="data-title">{{ row.name }}</div>
                  <div class="data-meta">
                    代码 <code class="inline-code">{{ row.code }}</code> · {{ row.startTime }}–{{ row.endTime }} · {{ row.durationMinutes }} 分钟
                  </div>
                </div>
                <div class="data-actions">
                  <van-button size="small" type="primary" plain hairline round @click="openShiftEditor(row)">编辑</van-button>
                </div>
              </div>
            </div>
          </section>
        </van-tab>

        <van-tab title="假期类型">
          <section class="card tab-card">
            <div class="head-row">
              <h3>假期类型</h3>
              <van-button size="small" type="primary" round @click="openLeaveEditor(null)">新增</van-button>
            </div>
            <p v-if="leaveError" class="error-block">{{ leaveError }}</p>
            <p v-else-if="!leaves.length" class="empty">暂无数据，请新增或检查数据库 <code>activity_types</code>。</p>
            <div v-else class="list-wrap">
              <div v-for="row in leaves" :key="row.code" class="catalog-row">
                <div class="data-main">
                  <div class="data-title">{{ row.code }} · {{ row.fullName }}</div>
                  <div class="data-sub">{{ row.description }}</div>
                </div>
                <div class="data-actions">
                  <van-button size="small" type="primary" plain hairline round @click="openLeaveEditor(row)">编辑</van-button>
                </div>
              </div>
            </div>
          </section>
        </van-tab>
      </van-tabs>
    </main>

    <van-popup v-model:show="showShiftEditor" position="bottom" round :style="{ height: '58%' }">
      <div class="popup-body">
        <h3>{{ shiftDraft.code ? '编辑班次' : '新增班次' }}</h3>
        <van-field v-model="shiftDraft.code" label="代码" placeholder="如 EARLY、OFFICE" :readonly="!!shiftOrigCode" />
        <van-field v-model="shiftDraft.name" label="名称" placeholder="如 早班、标准办公" />
        <van-field v-model="shiftDraft.startTime" label="开始" placeholder="09:00" />
        <van-field v-model="shiftDraft.endTime" label="结束" placeholder="18:00" />
        <van-field v-model.number="shiftDraft.durationMinutes" label="时长(分)" type="digit" placeholder="可空，将按起止推算" />
        <van-button type="primary" block round native-type="button" :loading="shiftSaving" @click="saveShift">保存</van-button>
      </div>
    </van-popup>

    <van-popup v-model:show="showLeaveEditor" position="bottom" round :style="{ height: '52%' }">
      <div class="popup-body">
        <h3>{{ leaveDraft.code ? '编辑假期' : '新增假期类型' }}</h3>
        <van-field v-model="leaveDraft.code" label="代码" placeholder="如 RDO、AL" :readonly="!!leaveOrigCode" />
        <van-field v-model="leaveDraft.fullName" label="全称" />
        <van-field v-model="leaveDraft.description" label="说明" type="textarea" rows="2" autosize />
        <van-button type="primary" block round native-type="button" :loading="leaveSaving" @click="saveLeave">保存</van-button>
      </div>
    </van-popup>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { showFailToast, showSuccessToast } from 'vant'
import {
  getAdminActivityTypes,
  getAdminShiftTypes,
  upsertAdminActivityType,
  upsertAdminShiftType,
} from '../../api'

const tabActive = ref(0)

const shifts = ref([])
const leaves = ref([])
const shiftError = ref('')
const leaveError = ref('')

const showShiftEditor = ref(false)
const shiftSaving = ref(false)
const shiftOrigCode = ref('')
const shiftDraft = ref({
  code: '',
  name: '',
  startTime: '',
  endTime: '',
  durationMinutes: 0,
})

const showLeaveEditor = ref(false)
const leaveSaving = ref(false)
const leaveOrigCode = ref('')
const leaveDraft = ref({
  code: '',
  fullName: '',
  description: '',
})

const durationFromRange = (start, end) => {
  const [sh, sm] = String(start).split(':').map(Number)
  const [eh, em] = String(end).split(':').map(Number)
  if (!Number.isFinite(sh) || !Number.isFinite(eh)) return 540
  return eh * 60 + em - (sh * 60 + sm)
}

const loadShifts = async () => {
  shiftError.value = ''
  try {
    const { data } = await getAdminShiftTypes()
    shifts.value = Array.isArray(data?.items) ? data.items : []
  } catch {
    shiftError.value = '加载班次失败（需管理员登录）'
    shifts.value = []
  }
}

const loadLeaves = async () => {
  leaveError.value = ''
  try {
    const { data } = await getAdminActivityTypes()
    leaves.value = Array.isArray(data?.items) ? data.items : []
  } catch {
    leaveError.value = '加载假期类型失败（需管理员登录）'
    leaves.value = []
  }
}

const refreshAll = async () => {
  await Promise.all([loadShifts(), loadLeaves()])
}

const openShiftEditor = (row) => {
  shiftOrigCode.value = row?.code || ''
  shiftDraft.value = row
    ? {
        code: row.code,
        name: row.name,
        startTime: row.startTime,
        endTime: row.endTime,
        durationMinutes: row.durationMinutes || durationFromRange(row.startTime, row.endTime),
      }
    : {
        code: '',
        name: '',
        startTime: '09:00',
        endTime: '18:00',
        durationMinutes: 540,
      }
  showShiftEditor.value = true
}

const saveShift = async () => {
  const code = shiftDraft.value.code.trim()
  const name = shiftDraft.value.name.trim()
  const startTime = shiftDraft.value.startTime.trim()
  const endTime = shiftDraft.value.endTime.trim()
  let durationMinutes = Number(shiftDraft.value.durationMinutes) || 0
  if (!code || !name || !startTime || !endTime) {
    showFailToast('请填写代码、名称与起止时间')
    return
  }
  if (durationMinutes <= 0) durationMinutes = durationFromRange(startTime, endTime)
  shiftSaving.value = true
  try {
    await upsertAdminShiftType({
      code,
      name,
      startTime,
      endTime,
      durationMinutes,
    })
    showSuccessToast('已保存班次')
    showShiftEditor.value = false
    await loadShifts()
  } catch {
    showFailToast('保存失败')
  } finally {
    shiftSaving.value = false
  }
}

const openLeaveEditor = (row) => {
  leaveOrigCode.value = row?.code || ''
  leaveDraft.value = row
    ? {
        code: row.code,
        fullName: row.fullName,
        description: row.description,
      }
    : {
        code: '',
        fullName: '',
        description: '',
      }
  showLeaveEditor.value = true
}

const saveLeave = async () => {
  const code = leaveDraft.value.code.trim()
  const fullName = leaveDraft.value.fullName.trim()
  const description = leaveDraft.value.description.trim()
  if (!code || !fullName) {
    showFailToast('请填写代码与全称')
    return
  }
  leaveSaving.value = true
  try {
    await upsertAdminActivityType({
      code,
      fullName,
      description: description || fullName,
    })
    showSuccessToast('已保存假期类型')
    showLeaveEditor.value = false
    await loadLeaves()
  } catch {
    showFailToast('保存失败')
  } finally {
    leaveSaving.value = false
  }
}

refreshAll()
</script>

<style scoped>
.screen-page {
  min-height: 100vh;
  background: var(--brand-bg);
  color: var(--brand-text);
}
.topbar {
  height: 52px;
  border-bottom: 1px solid var(--brand-border);
  background: var(--brand-card);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
}
.topbar h1 {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  color: var(--brand-title);
}
.tune-ico {
  color: var(--brand-primary-mid);
  opacity: 0.85;
}
.icon-btn {
  border: 0;
  background: transparent;
  padding: 6px;
  color: var(--brand-primary-mid);
}
.content {
  padding: 12px 12px 28px;
}
.lead {
  margin: 0 0 12px;
  font-size: 12px;
  line-height: 1.55;
  color: var(--brand-subtext);
}
.lead strong {
  color: var(--brand-title);
}
.lead code {
  font-size: 11px;
  background: var(--brand-primary-soft);
  padding: 1px 4px;
  border-radius: 4px;
}
.tab-card {
  margin-top: 8px;
}
.card {
  background: var(--brand-card);
  border: 1px solid var(--brand-border);
  border-radius: 12px;
  padding: 14px;
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.06);
}
.head-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.head-row h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--brand-title);
}
.error-block {
  color: var(--brand-danger);
  font-size: 13px;
  padding: 8px;
  background: #fef2f2;
  border-radius: 8px;
}
.empty {
  color: var(--brand-subtext);
  font-size: 13px;
  padding: 10px 0;
}
.list-wrap {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.catalog-row {
  border: 1px solid var(--brand-border);
  border-radius: 10px;
  padding: 12px;
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: flex-start;
  background: var(--brand-card);
}
.data-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--brand-title);
}
.data-meta {
  margin-top: 6px;
  font-size: 12px;
  color: var(--brand-subtext);
  line-height: 1.35;
}
.data-sub {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--brand-subtext);
  line-height: 1.4;
}
.inline-code {
  font-size: 11px;
  font-family: ui-monospace, monospace;
}
.data-actions {
  flex-shrink: 0;
}
.popup-body {
  padding: 16px;
}
.popup-body h3 {
  margin: 0 0 14px;
  font-size: 17px;
  font-weight: 700;
  color: var(--brand-title);
}
</style>
