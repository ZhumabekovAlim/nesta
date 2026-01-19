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
