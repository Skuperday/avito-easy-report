<template>
  <div>
    <h1 class="text-lg font-bold mb-4">Маппинг объявлений → объекты</h1>
    <p class="text-sm muted mb-4">Загрузите XLSX с колонками: Номер объявления, Объект. Маппинг применяется ко всем загружаемым отчётам.</p>

    <div class="flex gap-3 mb-6">
      <input type="file" accept=".xlsx" @change="uploadFile" class="text-sm" />
      <span v-if="uploadMsg" class="text-sm" :class="uploadOk ? 'text-green-400' : 'text-red-400'">{{ uploadMsg }}</span>
    </div>

    <div v-if="loading" class="text-center muted py-6">Загрузка...</div>

    <div v-else-if="mappings && Object.keys(mappings).length > 0" class="table-wrap">
      <table class="data-table compact">
        <thead><tr><th>Номер объявления</th><th>Объект</th><th></th></tr></thead>
        <tbody>
          <tr v-for="(obj, num) in mappings" :key="num">
            <td class="font-mono text-xs">{{ num }}</td>
            <td>{{ obj }}</td>
            <td><button class="btn secondary text-xs" @click="remove(num)">Удалить</button></td>
          </tr>
        </tbody>
      </table>
      <p class="text-xs muted mt-2">Всего: {{ Object.keys(mappings).length }}</p>
    </div>

    <div v-else class="text-center muted py-6">Маппингов пока нет. Загрузите файл.</div>

    <NuxtLink to="/" class="btn secondary text-sm mt-6 inline-block">← Назад</NuxtLink>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const auth = useAuth()

const mappings = ref<Record<string, string> | null>(null)
const loading = ref(true)
const uploadMsg = ref('')
const uploadOk = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await auth.apiFetch('/objects')
    const data = await res.json()
    mappings.value = data.mappings
  } catch(e: any) { uploadMsg.value = e.message; uploadOk.value = false }
  finally { loading.value = false }
}

async function uploadFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const form = new FormData()
  form.append('file', file)
  try {
    const res = await auth.apiFetch('/objects/upload', { method: 'POST', body: form })
    const data = await res.json()
    uploadMsg.value = `Загружено: ${data.count} (всего: ${data.total})`
    uploadOk.value = true
    load()
  } catch(e: any) { uploadMsg.value = e.message; uploadOk.value = false }
}

async function remove(num: string) {
  await auth.apiFetch(`/objects/${num}`, { method: 'DELETE' })
  load()
}

onMounted(load)
</script>
