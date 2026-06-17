<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="card w-full max-w-md">
      <div class="brand-mark">AE</div>
      <h1 class="mt-4 text-2xl font-black">Вход</h1>
      <p class="muted mt-2">Введите логин и пароль. После входа доступ определяется ролями учетной записи.</p>
      <div v-if="error" class="mt-4 rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{{ error }}</div>
      <div class="mt-5 grid gap-4">
        <div>
          <label class="mb-2 block text-sm font-medium">Логин</label>
          <input v-model="username" class="input" placeholder="например, admin" autocomplete="username" />
        </div>
        <div>
          <label class="mb-2 block text-sm font-medium">Пароль</label>
          <input v-model="password" class="input" type="password" placeholder="Введите пароль" autocomplete="current-password" />
        </div>
        <button class="btn" :disabled="loading" @click="handleLogin">{{ loading ? 'Вход...' : 'Войти' }}</button>
        <NuxtLink to="/register" class="btn secondary">Регистрация</NuxtLink>
      </div>
      <p class="muted mt-4 text-xs">Если ролей пока нет, после регистрации вы получите роль Guest без доступа к функционалу.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })
const auth = useAuth()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const handleLogin = async () => {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    await navigateTo('/')
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>
