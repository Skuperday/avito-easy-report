<template>
  <div>
    <div v-if="loading" class="text-center muted py-12">Загрузка...</div>
    <div v-else-if="error" class="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-200">{{ error }}</div>

    <div v-else v-for="r in data.reports" :key="r.reportId" class="mb-6">
      <h2 class="text-sm font-bold mb-2 muted">{{ r.fileName }}</h2>
      <div class="table-wrap">
        <table class="data-table compact">
          <thead>
            <tr>
              <th class="cursor-pointer select-none" @click="toggleSort('city')">
                Город <span class="muted">{{ sortIcon('city') }}</span>
              </th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('shows')">
                Показы <span class="muted">{{ sortIcon('shows') }}</span>
              </th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('views')">
                Просм. <span class="muted">{{ sortIcon('views') }}</span>
              </th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('contacts')">
                Контакты <span class="muted">{{ sortIcon('contacts') }}</span>
              </th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('ppConversion')">
                ПП% <span class="muted">{{ sortIcon('ppConversion') }}</span>
              </th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('pkConversion')">
                ПК% <span class="muted">{{ sortIcon('pkConversion') }}</span>
              </th>
              <th class="text-right cursor-pointer select-none" @click="toggleSort('avgViewPrice')">
                Ср. цена просм. <span class="muted">{{ sortIcon('avgViewPrice') }}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in sorted(r.stats)" :key="s.city">
              <td class="font-bold">{{ s.city }}</td>
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

    <div v-if="!loading && data.reports.length > 0" class="flex gap-3 pt-4">
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
const loading = ref(true)
const error = ref('')

const sortKey = ref('city')
const sortDir = ref<'asc' | 'desc'>('asc')

function toggleSort(key: string) {
  if (sortKey.value === key) sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  else { sortKey.value = key; sortDir.value = 'asc' }
}

function sortIcon(key: string) {
  if (sortKey.value !== key) return ''
  return sortDir.value === 'asc' ? '↑' : '↓'
}

function sorted(stats: any[]) {
  return [...stats].sort((a, b) => {
    const va = a[sortKey.value]
    const vb = b[sortKey.value]
    if (typeof va === 'string') return sortDir.value === 'asc' ? va.localeCompare(vb) : vb.localeCompare(va)
    return sortDir.value === 'asc' ? va - vb : vb - va
  })
}

function fmt(val: number | undefined, decimals = 0): string {
  if (val == null) return '0'
  return val.toLocaleString('ru-RU', { minimumFractionDigits: decimals, maximumFractionDigits: decimals })
}

onMounted(async () => {
  const ids = route.query.ids as string
  if (!ids) { error.value = 'Не выбраны отчёты'; loading.value = false; return }
  try {
    const res = await auth.apiFetch(`/reports/multi?ids=${ids}`)
    data.value = await res.json()
    if (!res.ok) error.value = data.value.error || 'Ошибка'
  } catch (e: any) { error.value = e.message }
  finally { loading.value = false }
})

const exportAll = () => window.open(`${config.public.apiBase}/export?token=${auth.token.value}`, '_blank')
</script>

<style>
.data-table.compact th,
.data-table.compact td {
  padding: 0.45rem 0.6rem;
  font-size: 0.8rem;
}
</style>
