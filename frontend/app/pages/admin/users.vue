<template>
  <div class="grid gap-4">
    <div v-if="error" class="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{{ error }}</div>
    <div v-if="success" class="rounded-xl border border-green-500/30 bg-green-500/10 p-3 text-sm text-green-200">{{ success }}</div>

    <div class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Имя</th>
            <th>Роль</th>
            <th class="text-right">Действия</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td class="muted text-xs">{{ u.id }}</td>
            <td class="font-bold">{{ u.username }}</td>
            <td>
              <span class="badge" :style="u.role === 'admin' ? { background: 'color-mix(in oklch, var(--chart-1) 18%, transparent)', color: 'var(--chart-1)' } : u.role === 'manager' ? { background: 'color-mix(in oklch, var(--chart-2) 18%, transparent)', color: 'var(--chart-2)' } : {}">{{ u.role }}</span>
            </td>
            <td class="text-right">
              <template v-if="u.id !== auth.user.value?.userId">
                <select
                  :value="u.role"
                  @change="updateRole(u.id, ($event.target as HTMLSelectElement).value)"
                  class="input inline-flex w-auto mr-2"
                  style="min-height:2rem;font-size:0.8rem;padding:0.25rem 0.5rem"
                >
                  <option value="guest">Guest</option>
                  <option value="manager">Manager</option>
                  <option value="admin">Admin</option>
                </select>
                <button @click="handleDelete(u.id, u.username)" class="btn destructive" style="min-height:2rem;font-size:0.8rem;padding:0.25rem 0.6rem">Удалить</button>
              </template>
              <span v-else class="muted text-xs">(вы)</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const auth = useAuth()
const users = ref<{ id: number; username: string; role: string }[]>([])
const error = ref('')
const success = ref('')

const fetchUsers = async () => { try { const r = await auth.apiFetch('/admin/users'); users.value = await r.json() } catch (e: any) { error.value = e.message } }
const updateRole = async (uid: number, role: string) => {
  error.value = ''; success.value = ''
  try {
    const r = await auth.apiFetch(`/admin/users/${uid}/role`, { method: 'PUT', body: JSON.stringify({ role }) })
    if (r.ok) { success.value = 'Роль обновлена'; setTimeout(() => success.value = '', 2000); await fetchUsers() }
    else { const d = await r.json(); error.value = d.error }
  } catch (e: any) { error.value = e.message }
}
const handleDelete = async (uid: number, username: string) => {
  if (!confirm(`Удалить «${username}»?`)) return
  error.value = ''
  try {
    const r = await auth.apiFetch(`/admin/users/${uid}`, { method: 'DELETE' })
    if (r.ok) { success.value = `«${username}» удалён`; setTimeout(() => success.value = '', 2000); await fetchUsers() }
    else { const d = await r.json(); error.value = d.error }
  } catch (e: any) { error.value = e.message }
}
onMounted(() => { if (!auth.isAdmin.value) { navigateTo('/'); return }; fetchUsers() })
</script>
