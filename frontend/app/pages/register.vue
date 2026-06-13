<template>
  <div class="min-h-screen grid place-items-center p-6">
    <div class="card w-full max-w-md">
      <div class="brand-mark">AE</div>
      <h1 class="mt-4 text-2xl font-black">Регистрация</h1>
      <p class="muted mt-2">Новая учетная запись создается с ролью Guest. Доступ к функционалу выдает администратор.</p>
      <div v-if="error" class="mt-4 rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{{ error }}</div>
      <div v-if="success" class="mt-4 rounded-xl border border-green-500/30 bg-green-500/10 p-3 text-sm text-green-200">{{ success }}</div>
      <div class="mt-5 grid gap-4">
        <div>
          <label class="mb-2 block text-sm font-medium">Логин</label>
          <input v-model="username" class="input" placeholder="например, petrov" autocomplete="username" />
        </div>
        <div>
          <label class="mb-2 block text-sm font-medium">Пароль</label>
          <input v-model="password" class="input" type="password" placeholder="Минимум 4 символа" autocomplete="new-password" />
        </div>
        <button class="btn" :disabled="loading" @click="handleRegister">{{ loading ? 'Регистрация...' : 'Зарегистрироваться' }}</button>
        <NuxtLink to="/login" class="btn secondary">Уже есть аккаунт</NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })
const auth = useAuth()
const username = ref('')
const password = ref('')
const error = ref('')
const success = ref('')
const loading = ref(false)

const handleRegister = async () => {
  error.value = ''
  success.value = ''
  loading.value = true
  try {
    await auth.register(username.value, password.value)
    success.value = 'Регистрация успешна! Перенаправляем...'
    setTimeout(() => navigateTo('/login'), 1500)
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>
