import { computed, ref } from 'vue'
import { getLocationCatalog } from '../api'

export const REGION_HOTEL_TITLE = {
  peninsula: '澳门美高梅（半岛店）',
  cotai: '美狮美高梅（氹仔店）',
}

const normalizeCatalogRow = (row) => ({
  slug: row.slug,
  category: row.category,
  region: row.region ?? '',
  title: row.title,
  subtitle: row.subtitle ?? '',
  detail: row.detail ?? '',
  sortOrder: row.sortOrder ?? row.sort_order ?? 0,
})

export function useLocationCatalog() {
  const catalogItems = ref([])
  const catalogLoaded = ref(false)

  const catalogBySlug = computed(() => {
    const map = new Map()
    catalogItems.value.forEach((row) => map.set(row.slug, normalizeCatalogRow(row)))
    return map
  })

  const peninsulaDining = computed(() =>
    catalogItems.value
      .filter((row) => row.category === 'dining_restaurant' && row.region === 'peninsula')
      .sort((a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0)),
  )

  const cotaiDining = computed(() =>
    catalogItems.value
      .filter((row) => row.category === 'dining_restaurant' && row.region === 'cotai')
      .sort((a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0)),
  )

  const loadLocationCatalog = async () => {
    try {
      const { data } = await getLocationCatalog()
      catalogItems.value = Array.isArray(data.items) ? data.items.map(normalizeCatalogRow) : []
    } catch {
      catalogItems.value = []
    } finally {
      catalogLoaded.value = true
    }
  }

  return {
    catalogItems,
    catalogLoaded,
    catalogBySlug,
    peninsulaDining,
    cotaiDining,
    loadLocationCatalog,
  }
}
