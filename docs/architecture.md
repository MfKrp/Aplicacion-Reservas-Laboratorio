# Arquitectura

La aplicación es un monolito modular Go. Los módulos separan interfaz web, transporte HTTP, dominio de reservas, consultas de inventario y persistencia SQLite.

## Componentes

- `cmd/server`: inicializa SQLite, construye repositorios y servicio, y arranca el servidor en `127.0.0.1:8080`.
- `internal/api`: adapta solicitudes/respuestas JSON y códigos HTTP. Recibe los servicios de reservas e inventario mediante interfaces; no accede directamente a SQLite.
- `internal/reservation/`: modelos y reglas de negocio para crear, consultar, cancelar y comprobar disponibilidad.
- `internal/inventory/`: catálogo inicial, validación y normalización de equipos, además de consultas y altas persistidas.
- `internal/storage/`: conexión, configuración, esquema y migraciones SQLite.
- `web/templates/index.html`: página que el servidor entrega en `/`.
- `web/static/`: CSS y JavaScript servidos bajo `/static/` con `http.FileServer` de la biblioteca estándar.
- `cmd/manual-check`: verificador local del servicio con una base independiente.
- `data/`: archivos de bases SQLite locales.

El navegador usa JavaScript nativo y Fetch API para comunicarse con la API en el mismo origen. La lógica de reservas permanece en el servicio; los handlers HTTP y la interfaz no duplican esas reglas.

## Interfaz

La página se organiza en inventario, alta de equipo, nueva reserva y el historial. Carga equipos y reservas al iniciar, permite registrar equipos, verificar disponibilidad, crear reservas y cancelar reservas activas; las canceladas permanecen en el historial. El inventario y el selector de reservas se vuelven a cargar después de un alta. El navegador limita los campos de fecha al máximo de seis meses calendario, y el servicio vuelve a validar el límite con la fecha actual del servidor. Las fechas elegidas se convierten a ISO/RFC3339 antes de enviarse.

El alta normaliza los espacios al inicio y al final de ID y nombre. `equipment.id` ya es clave primaria; SQLite rechaza de forma atómica un duplicado y no hace falta modificar el esquema ni migrar las bases existentes.

El máximo de reserva se calcula como la fecha actual del servidor más seis meses calendario. Si el día no existe en el mes destino, se ajusta al último día de ese mes; el máximo es inclusivo. Se comparan las fechas calendario en la zona horaria local del servidor.

## API REST

- `GET /api/equipment`: lista equipos.
- `POST /api/equipment`: crea un equipo recibiendo `id` y `name`.
- `GET /api/equipment/{id}/availability?starts_at=...&ends_at=...`: consulta disponibilidad mediante el servicio.
- `POST /api/reservations`: crea una reserva.
- `GET /api/reservations`: lista reservas activas y canceladas.
- `GET /api/reservations/{id}`: consulta una reserva.
- `PATCH /api/reservations/{id}/cancel`: cancela una reserva.

Las respuestas usan el envelope JSON `{ "data": ... }` en éxito y `{ "error": { "code": ..., "message": ... } }` en error. El servidor enlaza explícitamente a loopback y no escucha en interfaces de red externas.
