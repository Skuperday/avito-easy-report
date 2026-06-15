<template>
  <div>
    <div class="flex gap-2 mb-4">
      <NuxtLink to="/cabinets" class="btn secondary text-sm">← Назад</NuxtLink>
      <span class="text-sm font-bold self-center">{{ cabinet?.name || 'Кабинет' }}</span>
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
      <input ref="fileInput" type="file" accept=".xlsx" multiple class="hidden" @change="handleFileSelect" />
    </div>

    <!-- Toast -->
    <div class="fixed top-4 right-4 z-50 flex flex-col gap-2" style="max-width:360px">
      <div v-for="(n, i) in toasts" :key="i"
        class="rounded-xl border p-3 text-sm flex justify-between items-start gap-2 shadow-lg"
        :class="n.type === 'success' ? 'border-green-500/30 bg-green-500/10 text-green-300' : n.type === 'error' ? 'border-red-500/30 bg-red-500/10 text-red-300' : 'border-ring/30 bg-card text-foreground'"
        :style="{ backdropFilter: 'blur(12px)' }">
        <span class="whitespace-pre-wrap">{{ n.text }}</span>
        <button @click="toasts.splice(i, 1)" class="text-xs font-bold hover:opacity-70 ml-2 shrink-0 muted">✕</button>
      </div>
    </div>

    <!-- Список отчётов -->
    <div v-if="reports.length > 0" class="grid gap-2 mb-4">
      <div v-if="checked.size > 0" class="flex gap-2">
        <button class="btn destructive text-sm" @click="deleteSelected">Удалить ({{ checked.size }})</button>
        <button class="btn secondary text-sm" @click="checked = new Set(); checked = new Set()">Снять</button>
      </div>
      <div v-for="r in reports" :key="r.id"
        class="flex items-center gap-3 rounded-xl border p-3 text-sm"
        :style="{ borderColor: checked.has(r.id) ? 'var(--ring)' : 'var(--border)' }"
        :class="checked.has(r.id) ? 'bg-accent' : ''">
        <input type="checkbox" :checked="checked.has(r.id)" @change="toggleCheck(r.id)" class="w-4 h-4 accent-indigo-500" />
        <div class="flex-1 min-w-0"><div class="font-bold truncate">{{ r.fileName }}</div></div>
        <button @click="deleteSingle(r.id)" class="text-xs font-medium hover:underline" style="color: var(--destructive)">Удалить</button>
      </div>
    </div>
    <div v-else class="text-center muted text-sm py-8">Нет отчётов</div>

    <div class="flex gap-3 justify-center pt-2">
      <button class="btn" :disabled="checked.size < 2" @click="compareReports">Сравнить ({{ checked.size }})</button>
      <button class="btn secondary" :disabled="checked.size === 0" @click="showResults">Показать ({{ checked.size }})</button>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const auth = useAuth()
const route = useRoute()
const config = useRuntimeConfig()

const fileInput = ref<HTMLInputElement>()
const isDragging = ref(false)
const reports = ref<{ id: string; fileName: string }[]>([])
const checked = ref(new Set<string>())
const toasts = ref<{ type: string; text: string }[]>([])
const cabinet = ref<{ id: string; name: string } | null>(null)

const notify = (type: string, text: string) => { toasts.value.push({ type, text }); setTimeout(() => { if (toasts.value.length) toasts.value.shift() }, 6000) }

const uploadFile = async (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('cabinetId', route.params.id as string)
  notify('', `Загружаем ${file.name}...`)
  try {
    const res = await auth.apiFetch('/upload', { method: 'POST', body: formData, headers: {} })
    const data = await res.json()
    if (res.ok) {
      let msg = `✓ ${data.fileName} — ${data.rows} объявлений`
      if (data.warnings?.length) msg += '\n⚠ ' + data.warnings.join('\n⚠ ')
      notify('success', msg)
      await refresh()
    } else { notify('error', data.error) }
  } catch (e: any) { notify('error', e.message) }
}

const handleDrop = (e: DragEvent) => { isDragging.value = false; if (e.dataTransfer?.files) handleFiles(e.dataTransfer.files) }
const handleFileSelect = (e: Event) => { const files = (e.target as HTMLInputElement).files; if (files) handleFiles(files) }
const handleFiles = async (files: FileList) => { for (const f of files) { if (f.name.endsWith('.xlsx')) await uploadFile(f) } }

const refresh = async () => {
  try { const r = await auth.apiFetch('/cabinets/' + route.params.id + '/reports'); reports.value = await r.json() } catch {}
}
const toggleCheck = (id: string) => { checked.value.has(id) ? checked.value.delete(id) : checked.value.add(id); checked.value = new Set(checked.value) }
const deleteSingle = async (id: string) => { try { await auth.apiFetch('/reports/' + id, { method: 'DELETE' }); checked.value.delete(id); checked.value = new Set(checked.value); await refresh() } catch {} }
const deleteSelected = async () => { for (const id of checked.value) { try { await auth.apiFetch('/reports/' + id, { method: 'DELETE' }) } catch {} }; checked.value = new Set(); await refresh() }
const showResults = () => navigateTo('/results?ids=' + [...checked.value].join(','))
const compareReports = () => navigateTo('/results?ids=' + [...checked.value].join(',') + '&compare=1')

onMounted(async () => {
  await refresh()
  try { const r = await auth.apiFetch('/cabinets'); const list = await r.json(); cabinet.value = list.find((c: any) => c.id === route.params.id) } catch {}
})
</script>
