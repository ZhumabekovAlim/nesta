# nesta

1) Общая информация

Проект: сервис выноса мусора из квартир ЖК по подписке + интернет-магазин доп.товаров.
Назначение Backend: предоставить REST API для:

поиска/карточек ЖК и статусов обслуживания,

сбора заявок на запуск (с OTP),

управления подписками (создание, статусы, период),

ведения логов вывозов,

магазина (каталог, заказы, статусы),

платежей (инициация, webhooks),

админ-функций (CRUD сущностей, аудит).

2) Архитектура и технологии
2.1. Стек и подход

Язык: Golang

API: REST (JSON)

Архитектура: handlers → services → repositories

БД: PostgreSQL (рекомендуемо)

Миграции: goose / migrate / atlas (выбрать 1)

Логи: structured JSON (zap/zerolog), request_id/trace_id

Документация API: OpenAPI 3.0 (swagger)

2.2. Слои

Handlers: HTTP маршруты, auth middleware, DTO, статусы ответов.

Services: бизнес-логика (проверки, статусы, пороги, транзакции).

Repositories: SQL/ORM доступ к данным, транзакции.

2.3. Инфраструктурные требования

HTTPS обязательно (на уровне ingress/nginx).

Конфиги через env: DB_URL, JWT_SECRET, OTP_TTL, OTP_RATE_LIMIT, PAYMENT_PROVIDER_* и т.п.

Health endpoints: /health, /ready (проверка DB).

3) Роли, доступы, авторизация
3.1. Роли

guest: публичные методы (ЖК, тарифы, магазин просмотр), отправка/верификация OTP, заявка на запуск.

user: подписки, покупки, профиль, история, заказы.

admin: админ CRUD и операционные функции.

super_admin (опционально): управление админами/ролями/системными настройками.

3.2. Авторизация

OTP по телефону:

POST /api/v1/auth/otp/send

POST /api/v1/auth/otp/verify

Токены:

JWT access (короткий TTL)

refresh token (длинный TTL, хранить в БД, возможность отзыва)

RBAC:

Middleware проверяет роль и права доступа к /admin/*.

3.3. Безопасность OTP

TTL кода: напр. 3–5 минут

Rate-limit:

по телефону, по IP, по устройству (если есть device_id)

Защита от перебора: блокировка на N попыток на TTL.

4) Доменные сущности и бизнес-правила
4.1. ЖК (residential_complexes)

Статусы обслуживания:

ACTIVE

COLLECTING

PLANNED

NOT_SERVED

Порог запуска:

threshold_n (например 50)

current_requests (счетчик подтвержденных заявок)

Правило:

при достижении current_requests >= threshold_n → ЖК переводится в PLANNED (авто или вручную админом по настройке).

4.2. Заявка на запуск (complex_requests)

Уникальность: (complex_id, phone) — уникальный индекс

Поле verified должно становиться true только после OTP verify или отдельного подтверждения (если OTP flow сквозной).

При подтверждении заявки:

инкремент current_requests в ЖК (только если заявка впервые стала verified).

4.3. Тарифы (plans)

Поля: цена/мес, частота, лимиты (bags/day), описание, активность

Валидации: цена > 0, лимиты ≥ 0, name уникальный (желательно).

4.4. Подписка (subscriptions)

Привязка к:

user_id

complex_id

plan_id

address_json (дом/подъезд/этаж/квартира/доп.поля)

time_window (например "18:00-22:00" или start/end time)

instructions (текст)

Статусы:

PAYMENT_PENDING (если платеж обязателен)

ACTIVE

PAUSED (опционально)

CANCELED

EXPIRED

Правила:

Создать подписку можно только если ЖК = ACTIVE.

При отмене:

политика: либо сразу CANCELED, либо до конца оплаченного периода (зафиксировать как параметр сервиса).

История оплат хранится через payments (type=subscription).

4.5. Логи вывозов (pickup_logs)

Привязка: subscription_id

Статусы, минимум:

DONE

FAILED

SKIPPED (если нужно)

Поля: date, status, comment, optional причина (enum).

4.6. Магазин (products, orders, order_items)

Orders статусы:

NEW

PAID

ASSEMBLING

DELIVERING

DELIVERED

CANCELED

REFUNDED (опционально)

Правила:

Заказ создается из корзины (на фронте) как список items.

Цена позиции фиксируется в order_items.price на момент заказа.

Остатки:

на MVP: списание при переводе заказа в PAID или ASSEMBLING (зафиксировать правило).

защита от отрицательных остатков (transaction/locking).

4.7. Платежи (payments)

Поддержка 2 типов:

subscription

order

Поля:

provider

status: INIT, PENDING, PAID, FAILED, CANCELED, REFUNDED (минимум)

amount

payload_json (ответ/входные данные провайдера)

Webhook:

верификация подписи/секрета

идемпотентность по event_id/provider_payment_id

обработка переходов статусов и привязанных сущностей (order/subscription)

5) Требования к базе данных
5.1. Обязательные таблицы (минимум)

users

residential_complexes

complex_requests (unique complex_id + phone)

plans

subscriptions

pickup_logs

products

orders

order_items

payments

(админка) admins / roles / admin_sessions (или единая users+roles модель)

(желательно) audit_logs

5.2. Индексы (минимально)

residential_complexes(status, city, name)

complex_requests(complex_id, phone) unique

subscriptions(user_id), subscriptions(complex_id), subscriptions(status)

pickup_logs(subscription_id, date)

products(category_id), products(title)

orders(user_id, status, created_at)

payments(type, entity_id), unique(provider, provider_payment_id/event_id)

5.3. Транзакционность

Изменение статуса оплаты и связанной сущности (order/subscription) — в одной транзакции.

Инкремент current_requests и перевод статуса ЖК — атомарно (transaction).

6) API контракты (Backend)
6.1. Публичные/пользовательские

Комплексы

GET /api/v1/complexes?search=&status=&city=&only_active=

GET /api/v1/complexes/{id}

POST /api/v1/complexes/{id}/request
Создает/подтверждает заявку интереса (требование: OTP verified).

Планы

GET /api/v1/plans

Auth

POST /api/v1/auth/otp/send (phone)

POST /api/v1/auth/otp/verify (phone, code) → tokens

POST /api/v1/auth/refresh (refresh_token) → new access

POST /api/v1/auth/logout (revoke refresh)

Профиль

GET /api/v1/me

PATCH /api/v1/me (name, email?, default_address_json?)

Подписки

POST /api/v1/subscriptions
(plan_id, complex_id, address_json, time_window, instructions)
Ответ: subscription + payment_init (если нужно)

GET /api/v1/subscriptions/me

PATCH /api/v1/subscriptions/{id} (action=cancel/pause/resume по правилам)

GET /api/v1/pickups/{subscriptionId}

Магазин

GET /api/v1/products?category=&search=&in_stock=1

GET /api/v1/products/{id}

POST /api/v1/orders (items[], address_json, comment)

GET /api/v1/orders/me

GET /api/v1/orders/{id} (владелец)

Payments

POST /api/v1/payments/init (type=order/subscription, entity_id, provider)

POST /api/v1/payments/webhook/{provider} (подпись/секрет обязателен)

6.2. Админ (RBAC)

POST /api/v1/admin/login (если отдельный вход)

ЖК:

GET/POST/PATCH/DELETE /api/v1/admin/complexes

PATCH /api/v1/admin/complexes/{id}/status (если отдельный endpoint)

Заявки:

GET /api/v1/admin/complex-requests

Тарифы:

GET/POST/PATCH/DELETE /api/v1/admin/plans

Подписки:

GET /api/v1/admin/subscriptions

PATCH /api/v1/admin/subscriptions/{id} (force cancel/pause)

Логи вывозов:

GET/POST/PATCH /api/v1/admin/pickup-logs

POST /api/v1/admin/pickup-logs/import (опционально)

Магазин:

GET/POST/PATCH/DELETE /api/v1/admin/products

GET/PATCH /api/v1/admin/orders

Audit:

GET /api/v1/admin/audit-logs (желательно)

7) Форматы ответов и ошибки
7.1. Единый формат ошибки

code (строка)

message (человекочитаемо)

fields (опционально: ошибки валидации)

request_id

Примеры code:

VALIDATION_ERROR

UNAUTHORIZED

FORBIDDEN

NOT_FOUND

CONFLICT_DUPLICATE

PAYMENT_WEBHOOK_INVALID

RATE_LIMITED

7.2. Идемпотентность

Для orders и subscriptions (создание) — поддержать Idempotency-Key (заголовок) на уровне API gateway или сервиса (желательно).

Webhooks — обязателен idempotency по provider_event_id.

8) Логирование, аудит, мониторинг
8.1. Логи

request_id, user_id (если есть), path, status_code, latency_ms

ошибки сервиса (stack/message)

8.2. Audit log (желательно)

Фиксировать админ-действия:

кто (admin_id)

что сделал (entity, entity_id, action)

старое/новое значение (diff, json)

когда, ip

8.3. Метрики/алерты (минимум)

количество ошибок 5xx

latency p95

количество OTP send/verify (и rate-limited)

9) Нефункциональные требования

Производительность:

списки ЖК/товаров/заказов с пагинацией (limit/offset или cursor)

Безопасность:

HTTPS

JWT rotation (refresh)

rate-limit OTP

защита от дублей заявок на запуск

Совместимость:

версии API: /api/v1/...

Тестирование:

unit тесты сервисов

интеграционные тесты репозиториев (опционально)

10) MVP-границы Backend (чтобы не расползлось)

В MVP Backend обязательно:

complexes + statuses + search

complex_requests + OTP + уникальность + прогресс current/threshold

plans

subscriptions (создание, статусы, просмотр “мои”)

admin: ЖК/заявки/тарифы/подписки

payments: init + webhook (можно “ручной платеж” через админа, но модель payments оставить)

В этап 2:

полноценный магазин + остатки + админ склад

pickup_logs/маршруты

уведомления

---

## Быстрый старт (Docker)

```bash
docker compose up --build
```

После запуска API доступен на `http://localhost:8080`.

### Health/Ready

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Локальный запуск без Docker (опционально)

```bash
export DB_URL="postgres://nesta:nesta@localhost:5432/nesta?sslmode=disable"
export PORT=8080
export APP_ENV=development
export JWT_SECRET="dev-secret"
export ACCESS_TOKEN_TTL=15m
export REFRESH_TOKEN_TTL=720h
export OTP_TTL=5m
export OTP_RATE_LIMIT=1m
export OTP_MAX_ATTEMPTS=5

go run ./cmd/api
```

## Переменные окружения

- `DB_URL` — строка подключения к PostgreSQL.
- `PORT` — порт HTTP сервера.
- `APP_ENV` — окружение (`development` включает консольный логгер).
- `JWT_SECRET` — секрет для подписи JWT.
- `ACCESS_TOKEN_TTL` — TTL access токена (например `15m`).
- `REFRESH_TOKEN_TTL` — TTL refresh токена (например `720h`).
- `OTP_TTL` — время жизни OTP кода (например `5m`).
- `OTP_RATE_LIMIT` — ограничение отправки OTP по телефону (например `1m`).
- `OTP_MAX_ATTEMPTS` — максимум попыток ввода OTP.

## Миграции

В репозитории есть SQL миграция: `migrations/001_init.sql`. Формат совместим с `goose`.

Пример запуска (goose должен быть установлен локально):

```bash
goose -dir migrations postgres "$DB_URL" up
```

---

# Полное руководство по API для фронта (v1)

Ниже описаны **все доступные роуты** бэкенда, бизнес‑логика и примеры HTTP‑запросов/ответов. Формат — JSON. Версия API фиксируется префиксом `/api/v1`.

## Общие правила

### Авторизация
- Для защищённых эндпоинтов нужен заголовок:

```
Authorization: Bearer <access_token>
```

- Access‑token выдаётся после `OTP verify` и короткоживущий.
- Refresh‑token выдаётся вместе с access и хранится в БД. Его можно отозвать через `logout`.

### Формат ошибок
```json
{
  "code": "VALIDATION_ERROR",
  "message": "описание",
  "fields": {
    "field": "ошибка"
  },
  "request_id": "..."
}
```

### Пагинация
Во всех list‑эндпоинтах используется `limit` и `offset`.

---

## 1) Публичные эндпоинты

### 1.1. ЖК (комплексы)

**GET /api/v1/complexes** — список ЖК с фильтрами.

**Query params**:
- `search` — поиск по названию
- `status` — фильтр по статусу (ACTIVE, COLLECTING, PLANNED, NOT_SERVED)
- `city` — город
- `only_active` — `1` если нужен только `ACTIVE`
- `limit`, `offset`

**Пример:**
```bash
curl "http://localhost:8080/api/v1/complexes?search=park&city=msk&only_active=1&limit=20&offset=0"
```

**Ответ:**
```json
{
  "items": [
    {
      "ID": "c1",
      "Name": "Park View",
      "City": "msk",
      "Status": "ACTIVE",
      "Threshold": 50,
      "CurrentRequests": 18
    }
  ],
  "limit": 20,
  "offset": 0
}
```

**GET /api/v1/complexes/{id}** — карточка ЖК.

```bash
curl http://localhost:8080/api/v1/complexes/c1
```

---

### 1.2. Тарифы

**GET /api/v1/plans** — список активных тарифов.

```bash
curl http://localhost:8080/api/v1/plans
```

---

### 1.3. Магазин: каталог

**GET /api/v1/products** — список товаров.

**Query params**:
- `category` — id категории (если есть)
- `search` — строка поиска
- `in_stock=1` — только товары в наличии
- `limit`, `offset`

```bash
curl "http://localhost:8080/api/v1/products?search=мешок&in_stock=1"
```

**GET /api/v1/products/{id}** — карточка товара.

```bash
curl http://localhost:8080/api/v1/products/p1
```

---

## 2) Авторизация и OTP

### 2.1. Отправка OTP
**POST /api/v1/auth/otp/send**

**Body:**
```json
{ "phone": "+79990001122" }
```

**Бизнес‑логика**:
- OTP живёт `OTP_TTL` (по умолчанию 5 мин).
- Лимит отправки — `OTP_RATE_LIMIT` (по умолчанию 1 мин).
- При превышении лимита возвращается `RATE_LIMITED`.

**Ответ (dev):**
```json
{ "status": "sent", "expires_at": "2025-01-01T10:00:00Z", "dev_code": "123456" }
```

### 2.2. Верификация OTP
**POST /api/v1/auth/otp/verify**

**Body:**
```json
{ "phone": "+79990001122", "code": "123456" }
```

**Бизнес‑логика**:
- Код проверяется по хешу.
- Есть защита от перебора (`OTP_MAX_ATTEMPTS`).
- При успехе выдаются access и refresh токены.

**Ответ:**
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<refresh>",
  "expires_at": "2025-01-01T10:15:00Z"
}
```

### 2.3. Refresh токен
**POST /api/v1/auth/refresh**

**Body:**
```json
{ "refresh_token": "<refresh>" }
```

**Ответ:**
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<refresh>",
  "expires_at": "2025-01-01T10:15:00Z"
}
```

### 2.4. Logout
**POST /api/v1/auth/logout**

**Body:**
```json
{ "refresh_token": "<refresh>" }
```

---

## 3) Профиль пользователя

### 3.1. Получить профиль
**GET /api/v1/me**

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/me
```

### 3.2. Обновить профиль
**PATCH /api/v1/me**

**Body:**
```json
{
  "name": "Иван",
  "email": "ivan@mail.ru",
  "default_address_json": {"city": "Moscow", "street": "..."}
}
```

---

## 4) Заявки на запуск ЖК

### 4.1. Создать заявку
**POST /api/v1/complexes/{id}/request**

**Требует авторизации.**

**Body:**
```json
{ "phone": "+79990001122" }
```

**Бизнес‑логика**:
- Уникальность по `(complex_id, phone)`.
- Если заявка уже verified → ошибка.
- При успешной верификации увеличивается `current_requests`, при достижении `threshold_n` ЖК переводится в `PLANNED`.

---

## 5) Подписки

### 5.1. Создать подписку
**POST /api/v1/subscriptions**

**Body:**
```json
{
  "plan_id": "plan1",
  "complex_id": "c1",
  "address_json": {"house": "1", "flat": "12"},
  "time_window": "18:00-22:00",
  "instructions": "оставить у двери"
}
```

**Бизнес‑логика**:
- Создание только если ЖК = `ACTIVE`.
- Если план платный → статус `PAYMENT_PENDING`.

**Ответ:**
```json
{
  "subscription": {"ID": "s1", "Status": "PAYMENT_PENDING"},
  "payment_required": true
}
```

### 5.2. Мои подписки
**GET /api/v1/subscriptions/me**

### 5.3. Управление подпиской
**PATCH /api/v1/subscriptions/{id}**

**Body:**
```json
{ "action": "cancel" }
```

Доступные действия: `cancel`, `pause`, `resume`.

---

## 6) Логи вывозов

**GET /api/v1/pickups/{subscriptionId}** — список вывозов по подписке.

---

## 7) Магазин: заказы

### 7.1. Создать заказ
**POST /api/v1/orders**

**Body:**
```json
{
  "items": [
    {"product_id": "p1", "quantity": 2},
    {"product_id": "p2", "quantity": 1}
  ],
  "address_json": {"street": "..."},
  "comment": "позвонить за 30 минут"
}
```

**Бизнес‑логика**:
- Цена фиксируется в `order_items.price_cents`.
- Статус нового заказа: `NEW`.

### 7.2. Мои заказы
**GET /api/v1/orders/me**

### 7.3. Заказ по ID
**GET /api/v1/orders/{id}**

---

## 8) Платежи

### 8.1. Инициация платежа
**POST /api/v1/payments/init**

**Body:**
```json
{
  "type": "order",
  "entity_id": "o1",
  "provider": "stripe",
  "provider_payment_id": "pi_123",
  "amount_cents": 19900
}
```

### 8.2. Webhook от провайдера
**POST /api/v1/payments/webhook/{provider}**

**Body:**
```json
{
  "provider_payment_id": "pi_123",
  "status": "PAID",
  "payload": {"raw": "..."}
}
```

**Бизнес‑логика**:
- Webhook идемпотентен по `provider_payment_id`.
- При `PAID`:
  - `order` → статус `PAID` и списание остатков.
  - `subscription` → статус `ACTIVE`, выставление периода.

---

## 9) Админка (RBAC)

**Все /api/v1/admin/** эндпоинты требуют роль `admin`.

### 9.1. ЖК
- **GET /api/v1/admin/complexes** — список
- **POST /api/v1/admin/complexes** — создать
- **PATCH /api/v1/admin/complexes/{id}/status** — смена статуса

### 9.2. Тарифы
- **GET /api/v1/admin/plans**
- **POST /api/v1/admin/plans**
- **PATCH /api/v1/admin/plans/{id}**

### 9.3. Подписки
- **GET /api/v1/admin/subscriptions**
- **PATCH /api/v1/admin/subscriptions/{id}** (cancel/pause/resume)

### 9.4. Товары
- **GET /api/v1/admin/products**
- **POST /api/v1/admin/products**
- **PATCH /api/v1/admin/products/{id}**

### 9.5. Заказы
- **GET /api/v1/admin/orders**
- **PATCH /api/v1/admin/orders/{id}** (смена статуса)

### 9.6. Логи вывозов
- **POST /api/v1/admin/pickup-logs**
- **PATCH /api/v1/admin/pickup-logs/{id}**

---

## 10) Примеры ошибок

**OTP лимит:**
```json
{
  "code": "RATE_LIMITED",
  "message": "rate limited",
  "request_id": "..."
}
```

**Недостаточно прав:**
```json
{
  "code": "FORBIDDEN",
  "message": "insufficient permissions",
  "request_id": "..."
}
```

**Не найдено:**
```json
{
  "code": "NOT_FOUND",
  "message": "...",
  "request_id": "..."
}
```
