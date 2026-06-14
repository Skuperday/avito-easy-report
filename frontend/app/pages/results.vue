<template>
  <div>
    <div v-if="loading" class="text-center muted py-12">Загрузка...</div>
    <div v-else-if="error" class="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-200">{{ error }}</div>

    <!-- Переключатель группировки -->
    <div v-if="!loading && !error" class="flex gap-2 mb-4">
      <button class="btn text-sm" :class="groupBy === 'city' ? '' : 'secondary'" @click="groupBy = 'city'; reload()">По городам</button>
      <button class="btn text-sm" :class="groupBy === 'category' ? '' : 'secondary'" @click="groupBy = 'category'; reload()">По категориям</button>
      <button class="btn text-sm" :class="groupBy === 'name' ? '' : 'secondary'" @click="groupBy = 'name'; reload()">По заголовкам</button>
    </div>

    <!-- Сравнение периодов -->
    <template v-if="isCompare && compareData">
      <div class="table-wrap mb-4">
        <table class="data-table compact">
          <thead><tr>
            <th>{{ groupLabel }}</th>
            <th class="text-right" colspan="3">Показы</th>
            <th class="text-right" colspan="3">Просмотры</th>
            <th class="text-right" colspan="3">Контакты</th>
            <th class="text-right" colspan="3">ПП%</th>
            <th class="text-right" colspan="3">ПК%</th>
          </tr>
          <tr class="text-xs muted">
            <th></th>
            <th class="text-right">П1</th><th class="text-right">П2</th><th class="text-right">Δ</th>
            <th class="text-right">П1</th><th class="text-right">П2</th><th class="text-right">Δ</th>
            <th class="text-right">П1</th><th class="text-right">П2</th><th class="text-right">Δ</th>
            <th class="text-right">П1</th><th class="text-right">П2</th><th class="text-right">Δ</th>
            <th class="text-right">П1</th><th class="text-right">П2</th><th class="text-right">Δ</th>
          </tr></thead>
          <tbody>
            <tr v-for="(s, i) in compareData.delta" :key="s.key">
              <td class="font-bold">{{ s.key }}</td>
              <td class="text-right muted">{{ fmt(earlyRow(i)?.shows) }}</td>
              <td class="text-right">{{ fmt(lateRow(i)?.shows) }}</td>
              <td class="text-right" :style="{ color: s.shows >= 0 ? '#22c55e' : 'var(--destructive)' }">{{ s.shows >= 0 ? '+' : '' }}{{ fmt(s.shows) }}</td>
              <td class="text-right muted">{{ fmt(earlyRow(i)?.views) }}</td>
              <td class="text-right">{{ fmt(lateRow(i)?.views) }}</td>
              <td class="text-right" :style="{ color: s.views >= 0 ? '#22c55e' : 'var(--destructive)' }">{{ s.views >= 0 ? '+' : '' }}{{ fmt(s.views) }}</td>
              <td class="text-right muted">{{ fmt(earlyRow(i)?.contacts) }}</td>
              <td class="text-right">{{ fmt(lateRow(i)?.contacts) }}</td>
              <td class="text-right" :style="{ color: s.contacts >= 0 ? '#22c55e' : 'var(--destructive)' }">{{ s.contacts >= 0 ? '+' : '' }}{{ fmt(s.contacts) }}</td>
              <td class="text-right muted">{{ fmt(earlyRow(i)?.ppConversion, 1) }}%</td>
              <td class="text-right">{{ fmt(lateRow(i)?.ppConversion, 1) }}%</td>
              <td class="text-right">{{ s.ppConversion >= 0 ? '+' : '' }}{{ fmt(s.ppConversion, 1) }}%</td>
              <td class="text-right muted">{{ fmt(earlyRow(i)?.pkConversion, 1) }}%</td>
              <td class="text-right">{{ fmt(lateRow(i)?.pkConversion, 1) }}%</td>
              <td class="text-right">{{ s.pkConversion >= 0 ? '+' : '' }}{{ fmt(s.pkConversion, 1) }}%</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <!-- Обычный просмотр -->
    <template v-else>
      <div v-for="r in data.reports" :key="r.reportId" class="mb-6">
        <h2 class="text-sm font-bold mb-1">{{ r.fileName }}</h2>
        <p class="text-xs muted mb-3">
          {{ r.summary.totalShows.toLocaleString('ru') }} показов ·
          {{ r.summary.totalViews.toLocaleString('ru') }} просмотров ·
          {{ r.summary.totalContacts.toLocaleString('ru') }} контактов ·
          Топ городов: <span v-for="(t, i) in r.summary.topCities" :key="t.name">{{ t.name }}: {{ t.value }}<span v-if="i < r.summary.topCities.length-1">, </span></span>
        </p>

        <div class="table-wrap">
          <table class="data-table compact">
            <thead><tr>
              <th class="cursor-pointer select-none" @click="toggleSort('key')">{{ groupLabel }} <span class="muted">{{ sortIcon('key') }}</span></th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('shows')">Показы <span class="muted">{{ sortIcon('shows') }}</span></th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('views')">Просм. <span class="muted">{{ sortIcon('views') }}</span></th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('contacts')">Контакты <span class="muted">{{ sortIcon('contacts') }}</span></th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('ppConversion')">ПП% <span class="muted">{{ sortIcon('ppConversion') }}</span></th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('pkConversion')">ПК% <span class="muted">{{ sortIcon('pkConversion') }}</span></th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('avgViewPrice')">Ср. цена просм. <span class="muted">{{ sortIcon('avgViewPrice') }}</span></th>
            </tr></thead>
            <tbody>
              <tr v-for="s in sorted(r.stats)" :key="s.key">
                <td class="font-bold">{{ s.key }}</td>
                <td class="text-right">{{ fmt(s.shows) }}</td>
                <td class="text-right">{{ fmt(s.views) }}</td>
                <td class="text-right">{{ fmt(s.contacts) }}</td>
                <td class="text-right">{{ fmt(s.ppConversion, 1) }}%</td>
                <td class="text-right">{{ fmt(s.pkConversion, 1) }}%</td>
                <td class="text-right">{{ fmt(s.avgViewPrice, 2) }} ₽</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <div v-if="!loading && !error" class="flex gap-3 pt-4">
      <NuxtLink to="/" class="btn secondary text-sm">← Назад</NuxtLink>
      <button class="btn text-sm" @click="exportAll">Скачать XLSX</button>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const auth = useAuth()
const config = useRuntimeConfig()
const route = useRoute()

const data = ref<{ reports: any[] }>({ reports: [] })
const compareData = ref<any>(null)
const loading = ref(true)
const error = ref('')
const groupBy = ref('city')
const isCompare = ref(false)

const sortKey = ref('contacts')
const sortDir = ref<'asc' | 'desc'>('desc')
const groupLabel = computed(() => groupBy.value === 'city' ? 'Город' : groupBy.value === 'category' ? 'Категория' : 'Заголовок')

function toggleSort(key: string) { if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'; else { sortKey.value = key; sortDir.value = 'desc' } }
function sortIcon(key: string) { if (sortKey.value !== key) return ''; return sortDir.value === 'asc' ? '↑' : '↓' }
function sorted(stats: any[]) {
  return [...stats].sort((a, b) => {
    const va = a[sortKey.value], vb = b[sortKey.value]
    if (typeof va === 'string') return sortDir.value === 'asc' ? va.localeCompare(vb) : vb.localeCompare(va)
    return sortDir.value === 'asc' ? va - vb : vb - va
  })
}

async function reload() {
  const ids = route.query.ids as string
  if (!ids) return
  loading.value = true
  try {
    if (isCompare.value) {
      const res = await auth.apiFetch(`/reports/compare?ids=${ids}&groupBy=${groupBy.value}`)
      compareData.value = await res.json()
      if (!res.ok) error.value = compareData.value.error
    } else {
      const res = await auth.apiFetch(`/reports/multi?ids=${ids}&groupBy=${groupBy.value}`)
      data.value = await res.json()
      if (!res.ok) error.value = data.value.error
    }
  } catch (e: any) { error.value = e.message }
  finally { loading.value = false }
}

function fmt(val: number | undefined, decimals = 0): string {
  if (val == null) return '0'
  return val.toLocaleString('ru-RU', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
}

function earlyRow(i: number) { return compareData.value?.early?.stats?.[i] }
function lateRow(i: number) { return compareData.value?.late?.stats?.[i] }

onMounted(() => {
  isCompare.value = route.query.compare === '1'
  reload()
})

const exportAll = () => window.open(`${config.public.apiBase}/export?token=${auth.token.value}`, '_blank')
</script>

<style>
.data-table.compact th, .data-table.compact td { padding: 0.45rem 0.6rem; font-size: 0.8rem; }
</style>
