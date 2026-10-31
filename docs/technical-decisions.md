# Decisiones técnicas

## Confirmadas e implementadas

- **Lenguaje:** Go 1.27.1.
- **HTTP:** biblioteca estándar `net/http`, sin frameworks.
- **Dirección:** `127.0.0.1:8080`; los timeouts del servidor limitan esperas de lectura y escritura.
- **Persistencia:** SQLite mediante `database/sql` y `modernc.org/sqlite` v1.61.0.
- **Arquitectura:** monolito modular; los handlers reciben interfaces de servicio e inventario y no dependen de SQLite.
- **JSON de éxito:** envelope `{ "data": ... }`.
- **JSON de error:** envelope `{ "error": { "code": "...", "message": "..." } }`.
- **Tamaño máximo del cuerpo:** 1 MiB.
- **Solicitudes POST:** requieren `Content-Type: application/json`, rechazan campos desconocidos y aceptan exactamente un valor JSON.
- **Rutas no admitidas y métodos incorrectos:** responden JSON; los métodos incorrectos incluyen `Allow`.
- **Errores internos:** el cliente recibe un mensaje genérico HTTP 500; el detalle se registra localmente y no se devuelve información de SQLite.

## Endpoints y respuestas HTTP

| Método y ruta | Resultado exitoso | Errores principales |
|---|---|---|
| `GET /api/equipment` | 200, lista de equipos | 500 |
| `GET /api/equipment/{id}/availability` | 200, disponibilidad para el intervalo | 400, 404, 500 |
| `POST /api/reservations` | 201, reserva creada | 400, 404, 409, 413, 415, 500 |
| `GET /api/reservations` | 200, todas las reservas, incluidas las canceladas | 500 |
| `GET /api/reservations/{id}` | 200, reserva encontrada | 400, 404, 500 |
| `PATCH /api/reservations/{id}/cancel` | 200, reserva cancelada | 400, 404, 409, 500 |

Códigos generales: **400** solicitud, JSON, query o ID inválidos; **404** ruta, equipo o reserva inexistentes; **405** método no permitido; **409** conflicto de horario o estado; **413** cuerpo mayor a 1 MiB; **415** tipo de contenido distinto de JSON; **500** error interno sanitizado.

## Formatos JSON

Solicitud de creación:

```json
{
  "equipment_id": "MIC-01",
  "requester": "Ana Pérez",
  "starts_at": "2030-06-15T09:00:00-03:00",
  "ends_at": "2030-06-15T11:00:00-03:00"
}
```

Respuesta de éxito:

```json
{
  "data": {
    "id": 1,
    "equipment_id": "MIC-01",
    "requester": "Ana Pérez",
    "starts_at": "2030-06-15T12:00:00Z",
    "ends_at": "2030-06-15T14:00:00Z",
    "status": "ACTIVE",
    "created_at": "2026-10-31T12:00:00Z",
    "cancelled_at": null
  }
}
```

Respuesta de error:

```json
{
  "error": {
    "code": "reservation_conflict",
    "message": "equipment is already reserved for part of that interval"
  }
}
```

## Pendientes de revisión o definición

- Revisar si el envelope y los nombres de campos cumplen el contrato deseado para futuras integraciones.
- El formato RFC3339 de fechas sigue siendo la convención del dominio.
- No hay autenticación; el servidor solo enlaza a loopback.
- La interfaz web todavía no consume la API.