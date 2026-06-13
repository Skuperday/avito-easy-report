<template>
  <div>
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

    <!-- Статус -->
    <div v-if="uploadStatus" class="rounded-xl border p-3 text-sm whitespace-pre-wrap mb-4" :class="uploadStatus.type === 'success' ? 'border-green-500/30 bg-green-500/10 text-green-200' : 'border-red-500/30 bg-red-500/10 text-red-200'">
      {{ uploadStatus.text }}
    </div>

    <!-- Список отчётов с чекбоксами -->
    <div v-if="reports.length > 0" class="grid gap-2 mb-4">
      <div
        v-for="r in reports" :key="r.id"
        class="flex items-center gap-3 rounded-xl border p-3 text-sm"
        :style="{ borderColor: checked.has(r.id) ? 'var(--ring)' : 'var(--border)' }"
        :class="checked.has(r.id) ? 'bg-accent' : ''"
      >
        <input type="checkbox" :checked="checked.has(r.id)" @change="toggleCheck(r.id)" class="w-4 h-4 accent-indigo-500" />
        <div class="flex-1 min-w-0">
          <div class="font-bold truncate">{{ r.fileName }}</div>
          <div class="muted text-xs">{{ r.id.slice(0, 8) }}...</div>
        </div>
        <button @click="deleteReport(r.id)" class="text-xs font-medium hover:underline" style="color: var(--destructive)">Удалить</button>
      </div>
    </div>
    <div v-else class="text-center muted text-sm py-8">Нет загруженных отчётов</div>

    <!-- Кнопки -->
    <div class="flex gap-3 justify-center pt-2">
      <button class="btn" :disabled="checked.size === 0" @click="showResults">Показать результаты ({{ checked.size }})</button>
      <button v-if="reports.length > 0" class="btn secondary text-sm" @click="exportAll">Скачать XLSX</button>
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
const uploadStatus = ref<{ type: string; text: string } | null>(null)

const showStatus = (type: string, text: string) => {
  uploadStatus.value = { type, text }
  if (type === 'success') setTimeout(() => uploadStatus.value = null, 5000)
}

const uploadFile = async (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  showStatus('', `Загружаем ${file.name}...`)
  try {
    const res = await auth.apiFetch('/upload', { method: 'POST', body: formData, headers: {} })
    const data = await res.json()
    if (res.ok) {
      let msg = `✓ ${data.fileName} — ${data.rows} объявлений`
      if (data.warnings && data.warnings.length > 0) msg += `\n⚠ ${data.warnings.join('\n⚠ ')}`
      showStatus('success', msg)
      await refreshReports()
    } else { showStatus('error', `✗ ${data.error}`) }
  } catch (e: any) { showStatus('error', e.message) }
}

const handleDrop = (e: DragEvent) => { isDragging.value = false; if (e.dataTransfer?.files) handleFiles(e.dataTransfer.files) }
const handleFileSelect = (e: Event) => { const files = (e.target as HTMLInputElement).files; if (files) handleFiles(files) }
const handleFiles = async (files: FileList) => { for (const f of files) { if (f.name.endsWith('.xlsx')) await uploadFile(f) } }
const refreshReports = async () => { try { const r = await auth.apiFetch('/reports'); reports.value = await r.json() } catch {} }

const toggleCheck = (id: string) => {
  if (checked.value.has(id)) checked.value.delete(id)
  else checked.value.add(id)
  checked.value = new Set(checked.value)
}

const deleteReport = async (id: string) => {
  try {
    await auth.apiFetch(`/reports/${id}`, { method: 'DELETE' })
    checked.value.delete(id)
    checked.value = new Set(checked.value)
    await refreshReports()
  } catch {}
}

const showResults = () => {
  const ids = [...checked.value].join(',')
  navigateTo(`/results?ids=${ids}`)
}

const exportAll = () => window.open(`${config.public.apiBase}/export?token=${auth.token.value}`, '_blank')
onMounted(() => refreshReports())
</script>
