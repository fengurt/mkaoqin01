<template>
  <div class="screen-page">
    <header class="topbar">
      <button class="icon-btn" type="button" @click="$router.push('/me')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <h1>行程快捷配置</h1>
      <span class="material-symbols-outlined tune-ico">tune</span>
    </header>

    <main class="content">
      <section class="card">
        <div class="head-row">
          <h3>一级分组</h3>
          <van-button size="small" type="primary" round @click="openSectionEditor(null)">新增分组</van-button>
        </div>
        <p class="hint">
          一级标题为员工在「行程 → 新建事项」里看到的分组名称。每组绑定一类<strong>二级目录</strong>：酒店在岗、分区餐厅或快捷短语。保存后立即写入数据库，前台刷新即可生效。
        </p>
        <div v-if="sectionsLoadError" class="error-block">{{ sectionsLoadError }}</div>
        <div v-else-if="sections.length === 0" class="empty">暂无分组，请先新增。</div>
        <div v-else class="list-wrap">
          <div v-for="row in sections" :key="row.id" class="section-block">
            <div class="data-row">
              <div class="data-main">
                <div class="data-title">{{ row.sectionLabel }}</div>
                <div class="data-meta">
                  排序 {{ row.sortOrder }} · {{ categoryTitleZh(row.itemCategory) }} · {{ regionTitleZh(row.itemRegion) }}
                </div>
              </div>
              <div class="data-actions">
                <van-button size="small" type="primary" plain hairline round @click="openSectionEditor(row)">编辑</van-button>
                <van-button size="small" type="danger" plain hairline round @click="removeSection(row)">删除</van-button>
              </div>
            </div>
            <div class="secondary-preview">
              <span class="secondary-label">二级选项（当前库中共 {{ previewCount(row) }} 条）</span>
              <div v-if="previewItems(row).length" class="chip-row">
                <span v-for="item in previewItems(row)" :key="item.slug" class="chip">{{ item.title }}</span>
              </div>
              <p v-else class="secondary-empty">暂无匹配条目，请在下方「二级选项」中按类别新增，或检查区域筛选是否与条目一致。</p>
            </div>
          </div>
        </div>
      </section>

      <section class="card">
        <div class="head-row">
          <h3>二级选项（目录项）</h3>
          <van-button size="small" type="primary" round @click="openCatalogEditor(null)">新增条目</van-button>
        </div>
        <van-field
          :model-value="catalogFilterLabelZh"
          is-link
          readonly
          label="筛选类别"
          placeholder="请选择"
          @click="showCategoryPicker = true"
        />
        <div v-if="catalogLoadError" class="error-block">{{ catalogLoadError }}</div>
        <div v-else-if="catalogItems.length === 0" class="empty">该类别下暂无条目，请点击「新增条目」补充。</div>
        <div v-else class="list-wrap">
          <div v-for="row in catalogItems" :key="row.slug" class="catalog-row">
            <div class="data-main">
              <div class="data-title">{{ row.title }}</div>
              <p v-if="row.subtitle" class="data-sub">{{ row.subtitle }}</p>
              <div class="data-meta">
                {{ categoryTitleZh(row.category) }} · {{ regionTitleZh(row.region) }} · 排序 {{ row.sortOrder ?? 0 }}
              </div>
              <div class="data-code">内部编码 {{ row.slug }}</div>
            </div>
            <div class="data-actions">
              <van-button size="small" type="primary" plain hairline round @click="openCatalogEditor(row)">编辑</van-button>
              <van-button size="small" type="danger" plain hairline round @click="removeCatalog(row)">删除</van-button>
            </div>
          </div>
        </div>
      </section>
    </main>

    <van-popup v-model:show="showSectionEditor" position="bottom" round :style="{ height: '58%' }">
      <div class="popup-body">
        <h3>{{ sectionDraft.id ? '编辑分组' : '新增分组' }}</h3>
        <van-field v-model.number="sectionDraft.sortOrder" label="排序" type="digit" placeholder="数字越小越靠前" />
        <van-field v-model="sectionDraft.sectionLabel" label="一级标题" placeholder="例如：商务用餐 · 半岛店餐厅" />
        <van-field
          :model-value="categoryTitleZh(sectionDraft.itemCategory)"
          is-link
          readonly
          label="绑定目录类型"
          placeholder="请选择"
          @click="showSectionCategoryPicker = true"
        />
        <van-field
          v-model="sectionDraft.itemRegion"
          label="区域筛选"
          placeholder="留空=该类型下全部；餐厅填 peninsula（半岛）或 cotai（路氹）"
        />
        <van-button type="primary" block round native-type="button" @click="saveSection">保存</van-button>
      </div>
    </van-popup>

    <van-popup v-model:show="showCatalogEditor" position="bottom" round :style="{ height: '80%' }">
      <div class="popup-body popup-body--scroll">
        <h3>{{ catalogDraft.slugLocked ? '编辑条目' : '新增条目' }}</h3>
        <van-field
          v-model="catalogDraft.slug"
          label="内部编码"
          placeholder="英文与数字，如 dining_peninsula_x"
          :readonly="catalogDraft.slugLocked"
        />
        <van-field
          :model-value="categoryTitleZh(catalogDraft.category)"
          is-link
          readonly
          label="目录类型"
          @click="showCatalogCategoryPicker = true"
        />
        <van-field
          v-model="catalogDraft.region"
          label="所属区域"
          placeholder="餐厅必填：peninsula 或 cotai；其它类型可留空"
        />
        <van-field v-model="catalogDraft.title" label="显示名称" placeholder="员工看到的按钮文字" />
        <van-field v-model="catalogDraft.subtitle" label="副标题" placeholder="可选，用于备注亮点" />
        <van-field v-model="catalogDraft.detail" label="详细介绍" type="textarea" rows="4" autosize placeholder="可选，申报页「介绍」等场景可用" />
        <van-field v-model.number="catalogDraft.sortOrder" label="排序" type="digit" />
        <van-button type="primary" block round native-type="button" @click="saveCatalog">保存</van-button>
      </div>
    </van-popup>

    <van-action-sheet v-model:show="showCategoryPicker" :actions="categoryActions" cancel-text="取消" close-on-click-action @select="onPickCatalogFilter" />
    <van-action-sheet
      v-model:show="showSectionCategoryPicker"
      :actions="categoryActions"
      cancel-text="取消"
      close-on-click-action
      @select="onPickSectionCategory"
    />
    <van-action-sheet
      v-model:show="showCatalogCategoryPicker"
      :actions="categoryActions"
      cancel-text="取消"
      close-on-click-action
      @select="onPickCatalogCategory"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { showConfirmDialog, showFailToast, showSuccessToast } from 'vant'
import {
  deleteAdminLocationCatalog,
  deleteAdminScheduleQuickSection,
  getAdminLocationCatalog,
  getAdminScheduleQuickSections,
  upsertAdminLocationCatalog,
  upsertAdminScheduleQuickSection,
} from '../../api'

const categoryDefs = [
  {
    value: 'hotel_intro',
    title: '酒店 / 在岗地点',
    sheetSub: '对应行程「在岗」类按钮',
  },
  {
    value: 'dining_restaurant',
    title: '分区餐厅',
    sheetSub: '按半岛 / 路氹筛选',
  },
  {
    value: 'schedule_chip',
    title: '快捷短语',
    sheetSub: '如客户拜访、会议等',
  },
]

const categoryActions = categoryDefs.map((d) => ({
  name: d.title,
  subname: `${d.sheetSub}（${d.value}）`,
  value: d.value,
}))

const resolveCategoryValue = (action) =>
  action?.value ?? categoryDefs.find((d) => d.title === action?.name)?.value ?? ''

const categoryTitleZh = (value) => categoryDefs.find((x) => x.value === value)?.title || value || '—'

const REGION_ZH = {
  peninsula: '澳门半岛',
  cotai: '路氹（氹仔）',
}

const regionTitleZh = (region) => {
  const r = String(region || '').trim()
  if (!r) return '全部区域'
  return REGION_ZH[r] || r
}

const sections = ref([])
const sectionsLoadError = ref('')
const catalogItems = ref([])
const catalogLoadError = ref('')
const catalogCategoryFilter = ref('hotel_intro')
const allCatalogItems = ref([])

const catalogFilterLabelZh = computed(() => categoryTitleZh(catalogCategoryFilter.value))

const showSectionEditor = ref(false)
const showCatalogEditor = ref(false)
const showCategoryPicker = ref(false)
const showSectionCategoryPicker = ref(false)
const showCatalogCategoryPicker = ref(false)

const sectionDraft = reactive({
  id: 0,
  sortOrder: 0,
  sectionLabel: '',
  itemCategory: 'hotel_intro',
  itemRegion: '',
})

const catalogDraft = reactive({
  slugLocked: false,
  slug: '',
  category: 'hotel_intro',
  region: '',
  title: '',
  subtitle: '',
  detail: '',
  sortOrder: 0,
})

const previewItems = (section) => {
  const cat = section.itemCategory
  const reg = String(section.itemRegion || '').trim()
  return allCatalogItems.value
    .filter((it) => it.category === cat && (!reg || String(it.region || '').trim() === reg))
    .sort((a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0))
}

const previewCount = (section) => previewItems(section).length

const loadAllCatalogSnapshot = async () => {
  try {
    const { data } = await getAdminLocationCatalog()
    allCatalogItems.value = Array.isArray(data.items) ? data.items : []
  } catch {
    allCatalogItems.value = []
  }
}

const loadSections = async () => {
  sectionsLoadError.value = ''
  try {
    const { data } = await getAdminScheduleQuickSections()
    sections.value = Array.isArray(data.sections) ? data.sections : []
  } catch (err) {
    sections.value = []
    const status = err?.response?.status
    sectionsLoadError.value =
      status === 404
        ? '接口返回 404：后台进程可能仍为旧版本，请在项目根目录执行 bash scripts/dev-up.sh 重启全部服务后再试。'
        : err?.response?.data?.error || err?.message || '分组加载失败'
  }
}

const loadCatalog = async () => {
  catalogLoadError.value = ''
  try {
    const { data } = await getAdminLocationCatalog({ category: catalogCategoryFilter.value })
    catalogItems.value = Array.isArray(data.items) ? data.items : []
  } catch (err) {
    catalogItems.value = []
    const status = err?.response?.status
    catalogLoadError.value =
      status === 404
        ? '接口返回 404：请执行 bash scripts/dev-up.sh 重启后再试。'
        : err?.response?.data?.error || err?.message || '目录加载失败'
  }
}

const refreshAll = async () => {
  await Promise.all([loadSections(), loadCatalog(), loadAllCatalogSnapshot()])
}

const openSectionEditor = (row) => {
  if (row) {
    sectionDraft.id = row.id
    sectionDraft.sortOrder = row.sortOrder ?? 0
    sectionDraft.sectionLabel = row.sectionLabel || ''
    sectionDraft.itemCategory = row.itemCategory || 'hotel_intro'
    sectionDraft.itemRegion = row.itemRegion || ''
  } else {
    sectionDraft.id = 0
    sectionDraft.sortOrder = sections.value.length ? Math.max(...sections.value.map((s) => s.sortOrder ?? 0), 0) + 1 : 1
    sectionDraft.sectionLabel = ''
    sectionDraft.itemCategory = 'hotel_intro'
    sectionDraft.itemRegion = ''
  }
  showSectionEditor.value = true
}

const saveSection = async () => {
  try {
    const body = {
      sortOrder: Number(sectionDraft.sortOrder) || 0,
      sectionLabel: sectionDraft.sectionLabel.trim(),
      itemCategory: sectionDraft.itemCategory.trim(),
      itemRegion: sectionDraft.itemRegion.trim(),
    }
    if (sectionDraft.id) body.id = sectionDraft.id
    await upsertAdminScheduleQuickSection(body)
    showSuccessToast('已保存')
    showSectionEditor.value = false
    await refreshAll()
  } catch (err) {
    showFailToast(err?.response?.data?.error || '保存失败')
  }
}

const removeSection = async (row) => {
  try {
    await showConfirmDialog({ title: '删除分组', message: `确定删除「${row.sectionLabel}」？` })
  } catch {
    return
  }
  try {
    await deleteAdminScheduleQuickSection({ id: row.id })
    showSuccessToast('已删除')
    await refreshAll()
  } catch (err) {
    showFailToast(err?.response?.data?.error || '删除失败')
  }
}

const openCatalogEditor = (row) => {
  if (row) {
    catalogDraft.slugLocked = true
    catalogDraft.slug = row.slug || ''
    catalogDraft.category = row.category || 'hotel_intro'
    catalogDraft.region = row.region || ''
    catalogDraft.title = row.title || ''
    catalogDraft.subtitle = row.subtitle || ''
    catalogDraft.detail = row.detail || ''
    catalogDraft.sortOrder = row.sortOrder ?? 0
  } else {
    catalogDraft.slugLocked = false
    catalogDraft.slug = ''
    catalogDraft.category = catalogCategoryFilter.value || 'hotel_intro'
    catalogDraft.region = ''
    catalogDraft.title = ''
    catalogDraft.subtitle = ''
    catalogDraft.detail = ''
    catalogDraft.sortOrder = 0
  }
  showCatalogEditor.value = true
}

const saveCatalog = async () => {
  try {
    await upsertAdminLocationCatalog({
      slug: catalogDraft.slug.trim(),
      category: catalogDraft.category.trim(),
      region: catalogDraft.region.trim(),
      title: catalogDraft.title.trim(),
      subtitle: catalogDraft.subtitle.trim(),
      detail: catalogDraft.detail.trim(),
      sortOrder: Number(catalogDraft.sortOrder) || 0,
    })
    showSuccessToast('已保存')
    showCatalogEditor.value = false
    await refreshAll()
  } catch (err) {
    showFailToast(err?.response?.data?.error || '保存失败')
  }
}

const removeCatalog = async (row) => {
  try {
    await showConfirmDialog({
      title: '删除条目',
      message: `确定删除「${row.title}」？\n内部编码：${row.slug}`,
    })
  } catch {
    return
  }
  try {
    await deleteAdminLocationCatalog({ slug: row.slug })
    showSuccessToast('已删除')
    await refreshAll()
  } catch (err) {
    showFailToast(err?.response?.data?.error || '删除失败')
  }
}

const onPickCatalogFilter = (action) => {
  const v = resolveCategoryValue(action)
  if (v) catalogCategoryFilter.value = v
  showCategoryPicker.value = false
  loadCatalog()
}

const onPickSectionCategory = (action) => {
  const v = resolveCategoryValue(action)
  if (v) sectionDraft.itemCategory = v
  showSectionCategoryPicker.value = false
}

const onPickCatalogCategory = (action) => {
  const v = resolveCategoryValue(action)
  if (v) catalogDraft.category = v
  showCatalogCategoryPicker.value = false
}

onMounted(refreshAll)
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
  display: flex;
  flex-direction: column;
  gap: 12px;
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
.hint {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--brand-subtext);
  line-height: 1.55;
}
.hint strong {
  color: var(--brand-title);
  font-weight: 600;
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
.section-block {
  border: 1px solid var(--brand-border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--brand-surface);
}
.data-row {
  padding: 12px;
  display: flex;
  justify-content: space-between;
  gap: 8px;
  align-items: flex-start;
  background: var(--brand-card);
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
.data-sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--brand-subtext);
  line-height: 1.4;
}
.data-meta {
  margin-top: 6px;
  font-size: 12px;
  color: var(--brand-subtext);
  line-height: 1.35;
}
.data-code {
  margin-top: 4px;
  font-size: 11px;
  color: var(--brand-subtext);
  font-family: ui-monospace, monospace;
  opacity: 0.9;
}
.data-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex-shrink: 0;
}
.secondary-preview {
  padding: 10px 12px 12px;
  border-top: 1px dashed var(--brand-border);
  background: var(--brand-primary-soft);
}
.secondary-label {
  display: block;
  font-size: 11px;
  font-weight: 700;
  color: var(--brand-primary);
  letter-spacing: 0.02em;
  margin-bottom: 8px;
}
.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  font-size: 11px;
  font-weight: 600;
  padding: 5px 10px;
  border-radius: 999px;
  background: var(--brand-card);
  color: var(--brand-title);
  border: 1px solid var(--brand-border);
}
.secondary-empty {
  margin: 0;
  font-size: 12px;
  color: var(--brand-subtext);
  line-height: 1.45;
}
.popup-body {
  padding: 16px;
}
.popup-body--scroll {
  max-height: 85vh;
  overflow: auto;
}
.popup-body h3 {
  margin: 0 0 14px;
  font-size: 17px;
  font-weight: 700;
  color: var(--brand-title);
}
</style>
