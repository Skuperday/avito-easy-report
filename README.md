# Avito Easy Report

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![Nuxt](https://img.shields.io/badge/Nuxt-4-00DC82?logo=nuxt.js)](https://nuxt.com)
[![Docker](https://img.shields.io/badge/Docker-✓-2496ED?logo=docker)](https://docker.com)

Web-приложение для менеджеров Avito: загрузка XLSX-отчётов, агрегированная статистика по городам, конверсии и экспорт сводного Excel.

<p align="center">
  <img src="docs/screenshot.png" alt="Avito Easy Report" width="800">
</p>

## Возможности

- Drag-and-drop загрузка XLSX-отчётов из Avito Pro
- Автоматическая агрегация статистики по городам
- Конверсии: показы→просмотры, просмотры→контакты
- Средняя цена просмотра и контакта
- **Предупреждения** при отсутствии колонок в отчёте
- **Сортировка** результатов по любой колонке (ASC/DESC)
- Экспорт сводного `result.xlsx`
- **5 тем**: Светлая, Тёмная, Barbie, Twilight, Соник-X
- Система ролей: admin, manager, guest
- JWT-авторизация + админ-панель

## Быстрый старт

```bash
git clone https://github.com/Skuperday/avito-easy-report.git
cd avito-easy-report

# Создать .env
cp .env.example .env
# Сгенерировать JWT_SECRET:
# openssl rand -base64 32

# Запустить
docker compose up -d
```

Открыть **http://localhost:3000** — логин: `admin` / `admin`

## Разработка

### Бэкенд

```bash
go build -o avito-easy-server ./cmd/server/
PORT=8080 DB_HOST=localhost DB_USER=avito DB_PASSWORD=avito DB_NAME=avito ./avito-easy-server
```

### Фронтенд

```bash
cd frontend
npm install
NUXT_PUBLIC_API_BASE=http://localhost:8080/api npm run dev
```

## API

### Аутентификация
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/auth/login` | Вход → JWT |
| `POST` | `/api/auth/register` | Регистрация (роль: guest) |
| `GET` | `/api/auth/me` | Текущий пользователь |
| `GET` | `/health` | Health-check |

### Отчёты (JWT)
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/upload` | Загрузка XLSX (multipart) |
| `GET` | `/api/reports` | Список |
| `GET` | `/api/reports/multi?ids=...` | Статистика по нескольким |
| `GET` | `/api/reports/:id/stats` | Статистика по одному |
| `DELETE` | `/api/reports/:id` | Удалить |
| `GET` | `/api/export?token=...` | Скачать result.xlsx |

### Админка (admin)
| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/admin/users` | Список пользователей |
| `PUT` | `/api/admin/users/:id/role` | Сменить роль |
| `DELETE` | `/api/admin/users/:id` | Удалить |

## Структура проекта

```
avito-easy-report/
├── cmd/server/main.go           # Точка входа бэкенда
├── internal/
│   ├── config/config.go         # Конфигурация (env vars)
│   ├── database/db.go           # PostgreSQL + миграции + seed
│   ├── struct/models.go         # Модели данных + API-структуры
│   ├── service/
│   │   ├── reportService.go     # Парсинг XLSX + хранилище + статистика
│   │   └── authService.go       # JWT + пользователи
│   ├── handler/
│   │   ├── handler.go           # Отчёты API
│   │   └── authHandler.go       # Auth + Admin API
│   └── middleware/auth.go       # JWT middleware
├── frontend/                    # Nuxt 4 SPA
│   ├── app/
│   │   ├── pages/               # login, register, index (отчёты), results, admin/users
│   │   ├── layouts/default.vue  # App shell + sidebar + селектор тем
│   │   ├── composables/useAuth.ts
│   │   ├── middleware/auth.ts
│   │   ├── assets/css/main.css  # Дизайн-система (OKLCH, 4 темы)
│   │   └── types.ts
│   ├── nginx.conf
│   └── Dockerfile
├── docker-compose.yml
├── Dockerfile
├── .env.example
├── go.mod / go.sum
└── README.md
```

## Стек

| Слой | Технология |
|------|------------|
| Бэкенд | Go 1.26, Gin, GORM, PostgreSQL, JWT |
| Фронтенд | Nuxt 4, Vue 3, Tailwind CSS (OKLCH) |
| Инфра | Docker Compose, nginx |
| Excel | excelize v2 |

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `PORT` | 8080 | Порт бэкенда |
| `DB_HOST` | postgres | Хост PostgreSQL |
| `DB_PORT` | 5432 | Порт PostgreSQL |
| `DB_USER` | avito | Пользователь БД |
| `DB_PASSWORD` | avito | Пароль БД |
| `DB_NAME` | avito | Имя БД |
| `JWT_SECRET` | ... | Секрет для JWT |
| `CORS_ORIGIN` | * | Разрешённый origin |

## Лицензия

MIT
