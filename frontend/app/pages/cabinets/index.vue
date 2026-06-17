<template>
  <div>
    <div class="flex gap-2 mb-4">
      <input v-model="newName" class="input flex-1" placeholder="Название кабинета" @keyup.enter="createCabinet" />
      <button class="btn" @click="createCabinet">Создать</button>
    </div>

    <div v-if="cabinets.length > 0" class="grid gap-2">
      <div v-for="c in cabinets" :key="c.id" class="flex items-center justify-between rounded-xl border p-3 text-sm" style="border-color: var(--border)">
        <NuxtLink :to="'/cabinets/' + c.id" class="font-bold hover:underline">{{ c.name }}</NuxtLink>
        <button @click="deleteCabinet(c.id)" class="text-xs font-medium hover:underline" style="color: var(--destructive)">Удалить</button>
      </div>
    </div>
    <div v-else class="text-center muted text-sm py-8">Нет кабинетов. Создайте первый кабинет выше.</div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const auth = useAuth()
const cabinets = ref<{ id: string; name: string }[]>([])
const newName = ref('')

const fetch = async () => {
  try { const r = await auth.apiFetch('/cabinets'); cabinets.value = await r.json() } catch {}
}
const createCabinet = async () => {
  if (!newName.value.trim()) return
  try {
    await auth.apiFetch('/cabinets', { method: 'POST', body: JSON.stringify({ name: newName.value }) })
    newName.value = ''
    await fetch()
  } catch {}
}
const deleteCabinet = async (id: string) => {
  if (!confirm('Удалить кабинет и все его отчёты?')) return
  try { await auth.apiFetch('/cabinets/' + id, { method: 'DELETE' }); await fetch() } catch {}
}
onMounted(fetch)
</script>
