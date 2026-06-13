<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="sidebar-panel">
        <NuxtLink to="/" class="flex items-center gap-3 px-1 py-1">
          <div class="brand-mark">AE</div>
          <div>
            <div class="text-sm font-bold">Avito Easy</div>
            <div class="text-[0.65rem] muted">отчёты и статистика</div>
          </div>
        </NuxtLink>

        <nav class="grid gap-1">
          <NuxtLink to="/" class="nav-link" :class="{ 'router-link-active': route.path === '/' }">
            <span class="grid size-7 place-items-center rounded-lg bg-white/5 text-xs font-black uppercase">📊</span>
            <span class="min-w-0"><span class="block truncate">Отчёты</span></span>
          </NuxtLink>
          <NuxtLink v-if="auth.isAdmin.value" to="/admin/users" class="nav-link" :class="{ 'router-link-active': route.path === '/admin/users' }">
            <span class="grid size-7 place-items-center rounded-lg bg-white/5 text-xs font-black uppercase">👥</span>
            <span class="min-w-0"><span class="block truncate">Пользователи</span></span>
          </NuxtLink>
        </nav>

        <div class="mt-auto grid gap-3 border-t pt-4" style="border-color: var(--sidebar-border)">
          <div class="rounded-xl border p-3 text-sm" style="border-color: var(--sidebar-border); background: color-mix(in oklch, var(--background) 40%, transparent)">
            <div class="font-bold">{{ auth.user.value?.username || '?' }}</div>
            <div class="muted mt-1 text-xs">{{ auth.user.value?.role || 'guest' }}</div>
          </div>
          <button class="btn secondary text-sm" @click="auth.logout()">Выйти</button>
        </div>
      </div>
    </aside>

    <main class="main-inset">
      <div class="content-frame">
        <header class="topbar justify-between" style="min-height:2.5rem;padding:0.4rem 1rem">
          <span class="text-sm muted">{{ pageLabel }}</span>
          <select class="theme-select" :value="theme" @change="setTheme(($event.target as HTMLSelectElement).value)">
            <option value="light">☀ Светлая</option>
            <option value="dark">☾ Тёмная</option>
            <option value="barbie">🎀 Barbie</option>
            <option value="twilight">🌙 Twilight</option>
          </select>
        </header>

        <div class="page-container">
          <slot />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
const auth = useAuth()
const route = useRoute()
const theme = ref('dark')

function setTheme(name: string) {
  theme.value = name
  if (import.meta.client) {
    document.documentElement.setAttribute('data-theme', name)
    localStorage.setItem('theme', name)
  }
}

if (import.meta.client) {
  const saved = localStorage.getItem('theme') || 'dark'
  theme.value = saved
  document.documentElement.setAttribute('data-theme', saved)
}

const pageLabel = computed(() => {
  if (route.path === '/admin/users') return 'Учётные записи'
  if (route.path.startsWith('/results')) return 'Результаты'
  return 'Отчёты'
})
</script>
