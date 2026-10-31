# Arquitectura

La aplicación es un monolito modular Go. Los módulos separan transporte HTTP, dominio de reservas, consultas de inventario y persistencia SQLite.

## Componentes

- `cmd/server`: inicializa SQLite, construye los repositorios y el servicio, y arranca el servidor en `127.0.0.1:8080`.
- `internal/api`: adapta solicitudes/respuestas JSON y códigos HTTP. Recibe el servicio de reservas y el lector de inventario mediante interfaces; no accede directamente a SQLite.
- `internal/reservation/model.go`: modelos `Equipment` y `Reservation`, estados y errores del dominio.
- `internal/reservation/service.go`: validaciones y reglas de creación, consulta, disponibilidad y cancelación.
- `internal/reservation/repository.go`: operaciones SQL para persistir y consultar reservas.
- `internal/inventory/`: catálogo inicial y consultas SQL de equipos.
- `internal/storage/`: conexión, configuración, esquema y migraciones SQLite.
- `cmd/manual-check`: verificador local del servicio con una base independiente.
- `web/`: estructura preparada para la interfaz; todavía no está conectada a la API.
- `data/`: archivos de bases SQLite locales.

La lógica de negocio permanece en el servicio. Los handlers validan el formato del protocolo HTTP y delegan las operaciones al servicio.

## API REST

- `GET /api/equipment`: lista equipos.
- `GET /api/equipment/{id}/availability?starts_at=...&ends_at=...`: consulta disponibilidad mediante el servicio.
- `POST /api/reservations`: crea una reserva.
- `GET /api/reservations`: lista reservas activas y canceladas.
- `GET /api/reservations/{id}`: consulta una reserva.
- `PATCH /api/reservations/{id}/cancel`: cancela una reserva.

Las respuestas usan el envelope JSON `{ "data": ... }` en éxito y `{ "error": { "code": ..., "message": ... } }` en error. El servidor enlaza explícitamente a loopback; no escucha en interfaces de red externas.