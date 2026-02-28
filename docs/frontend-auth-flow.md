# Auth flow для фронтенда (обновленный)

## Новый сценарий входа

1. Пользователь вводит телефон.
2. Фронт вызывает `POST /api/v1/auth/phone/check`.
3. Если `requires_profile = true`:
   - показать форму с полями:
     - `name` (обязательно)
     - `email` (необязательно)
   - после заполнения вызвать `POST /api/v1/auth/otp/send` с `phone + name (+ email)`.
4. Если `requires_profile = false`:
   - сразу вызвать `POST /api/v1/auth/otp/send` только с `phone`.
5. Пользователь вводит OTP.
6. Фронт вызывает `POST /api/v1/auth/otp/verify`.
7. В ответ получает `access_token`, `refresh_token`, `expires_at` и выполняет вход.

---

## API контракты

### 1) Проверка телефона (без отправки SMS)

`POST /api/v1/auth/phone/check`

Request:
```json
{
  "phone": "+79990001122"
}
```

Response 200:
```json
{
  "phone": "79990001122",
  "is_new_user": true,
  "requires_profile": true
}
```

### 2) Отправка OTP

`POST /api/v1/auth/otp/send`

#### Для нового пользователя
```json
{
  "phone": "+79990001122",
  "name": "Иван Иванов",
  "email": "ivan@example.com"
}
```
`name` обязателен, `email` необязателен.

#### Для существующего пользователя
```json
{
  "phone": "+79990001122"
}
```

Response 200:
```json
{
  "status": "sent",
  "expires_at": "2026-01-01T12:00:00Z",
  "dev_code": "123456"
}
```

> `dev_code` возвращается только в dev/echo режимах бэкенда.

### 3) Подтверждение OTP

`POST /api/v1/auth/otp/verify`

Request:
```json
{
  "phone": "+79990001122",
  "code": "123456"
}
```

Response 200:
```json
{
  "access_token": "...",
  "refresh_token": "...",
  "expires_at": "2026-01-01T12:30:00Z"
}
```

---

## Что поменять на фронте

- Разбить текущий шаг "ввод телефона -> сразу send otp" на два шага:
  - `phone/check`
  - условный `otp/send`
- Добавить промежуточный экран/модалку профиля для новых пользователей.
- Валидировать `name` как обязательное поле только когда `requires_profile = true`.
- Для существующих пользователей отправлять `otp/send` без `name/email`.
- Текущий экран ввода OTP и шаг `otp/verify` остаются без изменений.
