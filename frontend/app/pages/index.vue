<template>
  <div>
    <!-- Выбор типа отчёта -->
    <div class="flex items-center gap-3 mb-4">
      <span class="text-sm muted">Тип отчёта:</span>
      <select v-model="reportType" class="theme-select">
        <option value="avito">🏷 Avito (объявления)</option>
        <option value="hr">💼 HR (вакансии)</option>
      </select>
    </div>

    <!-- Drop zone -->
    <div
      class="border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition hover:opacity-90 mb-4"
      :class="isDragging ? 'border-ring bg-muted/10' : ''"
      :style="{ borderColor: isDragging ? 'var(--ring)' : 'var(--border)' }"
      @click="fileInput?.click()"
      @dragover.prevent="isDragging = true"
      @dragleave="isDragging = false"
      @drop.prevent="handleDrop"
    >
      <div class="text-3xl mb-2">📊</div>
      <p class="text-sm muted">Перетащите XLSX-отчёты</p>
      <p class="text-xs muted mt-1">или нажмите для выбора</p>
      <input ref="fileInput" type="file" accept=".xlsx" multiple class="hidden" @change="handleFileSelect" />
    </div>

    <!-- Список отчётов -->
    <div v-if="reports.length > 0" class="mb-4">
      <div class="flex gap-2 mb-1" v-if="checked.size > 0">
        <button class="btn destructive text-sm" @click="deleteSelected">Удалить ({{ checked.size }})</button>
        <button class="btn secondary text-sm" @click="checked = new Set(); checked = new Set()">Снять</button>
      </div>
      <div v-for="r in reports" :key="r.id">
        <div
          class="flex items-center gap-3 rounded-xl border p-3 text-sm cursor-pointer"
          :style="{ borderColor: checked.has(r.id) ? 'var(--ring)' : expandedId === r.id ? 'var(--ring)' : 'var(--border)' }"
          :class="checked.has(r.id) ? 'bg-accent' : ''"
          @click="toggleExpand(r.id)"
        >
          <input type="checkbox" :checked="checked.has(r.id)" @change="toggleCheck(r.id)" @click.stop class="w-4 h-4 accent-indigo-500 shrink-0" />
          <div class="flex-1 min-w-0"><div class="font-bold truncate">{{ r.fileName }}</div><div class="muted text-xs">{{ r.id.slice(0, 8) }}...</div></div>
          <button @click.stop="deleteSingle(r.id)" class="text-xs font-medium hover:underline shrink-0" style="color: var(--destructive)">Удалить</button>
        </div>
        <!-- Развёрнутая статистика -->
        <div v-if="expandedId === r.id" class="border border-t-0 rounded-b-xl px-4 py-3 mb-1 text-sm" style="border-color: var(--ring); background: color-mix(in oklch, var(--ring) 4%, transparent)">
          <div v-if="expandedLoading" class="muted text-xs">Загрузка...</div>
          <div v-else-if="expandedStats">
            <p class="text-xs muted mb-2">
              {{ expandedStats.totalShows?.toLocaleString('ru') }} показов ·
              {{ expandedStats.totalViews?.toLocaleString('ru') }} просмотров ·
              {{ expandedStats.totalContacts?.toLocaleString('ru') }} контактов ·
              {{ expandedStats.totalExpense?.toLocaleString('ru', {minimumFractionDigits: 2}) }} ₽ расходов
            </p>
            <div class="flex flex-wrap gap-1.5 text-xs" v-if="expandedStats.topCities?.length">
              <span class="muted mr-1">Топ городов:</span>
              <span class="badge" v-for="t in expandedStats.topCities" :key="t.name">{{ t.name }}: {{ t.value }}</span>
            </div>
            <div class="flex flex-wrap gap-1.5 text-xs mt-1" v-if="expandedStats.topOffers?.length">
              <span class="muted mr-1">Топ объявлений:</span>
              <span class="badge" v-for="t in expandedStats.topOffers" :key="t.name">{{ t.name.slice(0, 30) }}: {{ t.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="text-center muted text-sm py-8">Нет загруженных отчётов</div>

    <div class="flex gap-3 justify-center pt-2">
      <button class="btn" :disabled="checked.size < 2" @click="compareReports">Сравнить ({{ checked.size }})</button>
      <button class="btn secondary" :disabled="checked.size === 0" @click="showResults">Показать ({{ checked.size }})</button>
      <button v-if="reports.length > 0" class="btn secondary text-sm" @click="exportAll">Скачать XLSX</button>
    </div>

    <!-- Toast -->
    <div class="fixed top-4 right-4 z-50 flex flex-col gap-1.5" style="max-width:300px">
      <div v-for="(n, i) in toasts" :key="i"
        class="rounded-lg border px-2.5 py-1.5 text-xs flex items-center gap-2 shadow-lg"
        :class="n.type === 'success' ? 'border-green-500/30 bg-green-500/10 text-green-300' : n.type === 'error' ? 'border-red-500/30 bg-red-500/10 text-red-300' : 'border-ring/30 bg-card text-foreground'">
        <span class="shrink-0">{{ n.type === 'success' ? '✓' : n.type === 'error' ? '✗' : '·' }}</span>
        <span class="truncate flex-1">{{ n.text }}</span>
        <button @click="toasts.splice(i, 1)" class="shrink-0 hover:opacity-70 muted text-xs">✕</button>
      </div>
      <button v-if="toasts.length > 1" @click="toasts = []" class="text-xs muted hover:text-foreground self-end">закрыть все</button>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const auth = useAuth()
const config = useRuntimeConfig()

const fileInput = ref<HTMLInputElement>()
const isDragging = ref(false)
const reports = ref<{ id: string; fileName: string }[]>([])
const checked = ref(new Set<string>())
const toasts = ref<{ type: string; text: string }[]>([])
const reportType = ref('avito')
const expandedId = ref<string | null>(null)
const expandedStats = ref<any>(null)
const expandedLoading = ref(false)

const notify = (type: string, text: string) => {
  toasts.value.push({ type, text })
}

const toggleExpand = async (id: string) => {
  if (expandedId.value === id) {
    expandedId.value = null
    expandedStats.value = null
    return
  }
  expandedId.value = id
  expandedLoading.value = true
  expandedStats.value = null
  try {
    const res = await auth.apiFetch(`/reports/${id}/stats?groupBy=city`)
    if (res.ok) {
      const data = await res.json()
      expandedStats.value = data.summary
    }
  } catch {}
  expandedLoading.value = false
}

const uploadFile = async (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  notify('', `Загружаем ${file.name}...`)
  try {
    const res = await auth.apiFetch('/upload', { method: 'POST', body: formData, headers: {} })
    const data = await res.json()
    if (res.ok) {
      const shortName = data.fileName.length > 40 ? data.fileName.slice(0, 37) + '...' : data.fileName
      let msg = `${shortName} — ${data.rows} строк`
      if (data.warnings?.length) msg += ` (${data.warnings.length} колонок не найдено)`
      notify('success', msg)
      await refreshReports()
    } else { notify('error', data.error) }
  } catch (e: any) { notify('error', e.message) }
}

const handleDrop = (e: DragEvent) => { isDragging.value = false; if (e.dataTransfer?.files) handleFiles(e.dataTransfer.files) }
const handleFileSelect = (e: Event) => { const files = (e.target as HTMLInputElement).files; if (files) handleFiles(files) }
const handleFiles = async (files: FileList) => { for (const f of files) { if (f.name.endsWith('.xlsx')) await uploadFile(f) } }
const refreshReports = async () => { try { const r = await auth.apiFetch('/reports'); reports.value = await r.json() } catch {} }
const toggleCheck = (id: string) => { checked.value.has(id) ? checked.value.delete(id) : checked.value.add(id); checked.value = new Set(checked.value) }

const deleteSingle = async (id: string) => {
  try { await auth.apiFetch(`/reports/${id}`, { method: 'DELETE' }); checked.value.delete(id); checked.value = new Set(checked.value); await refreshReports() } catch {}
}
const deleteSelected = async () => {
  for (const id of checked.value) { try { await auth.apiFetch(`/reports/${id}`, { method: 'DELETE' }) } catch {} }
  checked.value = new Set(); await refreshReports()
}

const showResults = () => navigateTo(`/results?ids=${[...checked.value].join(',')}`)
const compareReports = () => navigateTo(`/results?ids=${[...checked.value].join(',')}&compare=1`)
const exportAll = () => window.open(`${config.public.apiBase}/export?token=${auth.token.value}`, '_blank')
onMounted(() => refreshReports())
</script>
