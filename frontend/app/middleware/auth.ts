// Middleware: защита маршрутов
export default defineNuxtRouteMiddleware((to) => {
  // Публичные страницы
  if (to.path === '/login' || to.path === '/register') return

  // Проверка токена
  const token = import.meta.client ? localStorage.getItem('token') : null
  if (!token) {
    return navigateTo('/login')
  }
})
