# Arquitectura

La aplicación es un monolito modular Go. Los módulos separan interfaz web, transporte HTTP, dominio de reservas, consultas de inventario y persistencia SQLite.

## Componentes

- `cmd/server`: inicializa SQLite, construye repositorios y servicio, y arranca el servidor en `127.0.0.1:8080`.
- `internal/api`: adapta solicitudes/respuestas JSON y códigos HTTP. Recibe el servicio de reservas y el lector de inventario mediante interfaces; no accede directamente a SQLite.
- `internal/reservation/`: modelos y reglas de negocio para crear, consultar, cancelar y comprobar disponibilidad.
- `internal/inventory/`: catálogo inicial y consultas SQL de equipos.
- `internal/storage/`: conexión, configuración, esquema y migraciones SQLite.
- `web/templates/index.html`: página que el servidor entrega en `/`.
- `web/static/`: CSS y JavaScript servidos bajo `/static/` con `http.FileServer` de la biblioteca estándar.
- `cmd/manual-check`: verificador local del servicio con una base independiente.
- `data/`: archivos de bases SQLite locales.

El navegador usa JavaScript nativo y Fetch API para comunicarse con la API en el mismo origen. La lógica de reservas permanece en el servicio; los handlers HTTP y la interfaz no duplican esas reglas.

## Interfaz

La página se organiza en inventario, nueva reserva y el historial. Carga equipos y reservas al iniciar, permite verificar disponibilidad, crear reservas y cancelar reservas activas; las canceladas permanecen en el historial. Las fechas elegidas en el navegador se convierten a ISO/RFC3339 antes de enviarse.

## API REST

- `GET /api/equipment`: lista equipos.
- `GET /api/equipment/{id}/availability?starts_at=...&ends_at=...`: consulta disponibilidad mediante el servicio.
- `POST /api/reservations`: crea una reserva.
- `GET /api/reservations`: lista reservas activas y canceladas.
- `GET /api/reservations/{id}`: consulta una reserva.
- `PATCH /api/reservations/{id}/cancel`: cancela una reserva.

Las respuestas usan el envelope JSON `{ "data": ... }` en éxito y `{ "error": { "code": ..., "message": ... } }` en error. El servidor enlaza explícitamente a loopback y no escucha en interfaces de red externas.