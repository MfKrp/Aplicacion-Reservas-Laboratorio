# Decisiones técnicas

## Confirmadas e implementadas

- **Lenguaje y HTTP:** Go 1.27.1 con `net/http`, sin frameworks.
- **Dirección:** el servidor escucha en `127.0.0.1:8080`, accesible desde `http://localhost:8080`; usa timeouts de lectura y escritura.
- **Persistencia:** SQLite mediante `database/sql` y `modernc.org/sqlite` v1.61.0.
- **Interfaz:** HTML5, CSS3 y JavaScript nativo; no se agregan frameworks ni dependencias externas.
- **Archivos estáticos:** `GET /` sirve `web/templates/index.html`; `/static/` se resuelve con `http.FileServer(http.Dir("web/static"))`.
- **Comunicación web/API:** Fetch API y rutas relativas `/api/...`, por lo que navegador y API comparten origen y no requieren CORS.
- **Dependencias:** handlers reciben interfaces del servicio y del inventario; no tienen acceso directo a SQLite.
- **Diseño:** tres secciones (inventario, nueva reserva e historial), diseño adaptable, botones de aceptación en `#8D15ED` y sin animaciones ni recursos externos.
- **Fechas de formulario:** el navegador recibe fecha/hora local; JavaScript las convierte a ISO/RFC3339 UTC antes de consultar disponibilidad o crear reservas.
- **Respuestas API:** envelope `{ "data": ... }` en éxito y `{ "error": { "code": "...", "message": "..." } }` en error.
- **Seguridad de contenido:** datos de inventario e historial se insertan usando `textContent`, no HTML construido con datos de usuario.

## Endpoints utilizados por la interfaz

| Método | Ruta | Función en la interfaz |
|---|---|---|
| GET | `/api/equipment` | Cargar inventario y opciones del formulario. |
| GET | `/api/equipment/{id}/availability` | Consultar disponibilidad con `starts_at` y `ends_at`. |
| POST | `/api/reservations` | Crear reserva con `equipment_id`, `requester`, `starts_at`, `ends_at`. |
| GET | `/api/reservations` | Cargar historial, incluidas canceladas. |
| PATCH | `/api/reservations/{id}/cancel` | Cancelar tras confirmación del usuario. |

La API sigue siendo la autoridad final para validar reglas y conflictos.

## Códigos HTTP

- **200:** consultas y cancelación correcta.
- **201:** reserva creada.
- **400:** solicitud, JSON, query o ID inválidos.
- **404:** ruta, equipo o reserva inexistentes.
- **405:** método no permitido.
- **409:** conflicto de horario o estado.
- **413:** cuerpo mayor a 1 MiB.
- **415:** tipo de contenido distinto de JSON.
- **500:** error interno sanitizado, sin detalles SQLite.

## Pendientes de revisión

- Revisar el aspecto visual y los textos con usuarios del laboratorio.
- Confirmar si la selección local de zona horaria y la visualización en hora local son adecuadas para los equipos de uso.
- La interfaz no incorpora autenticación; el servidor solo enlaza a loopback.