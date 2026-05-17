package httpapi

// HeaderConsoleUserID — внутренний заголовок к processing-service: владелец данных консоли (authn.console_users.id).
// Устанавливается только api-service; processing не доверяет внешним клиентам при проксировании через API.
const HeaderConsoleUserID = "X-ASOC-Console-User-ID"
