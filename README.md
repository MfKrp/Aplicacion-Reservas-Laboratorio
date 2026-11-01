# Sistema de Reservas de Equipamiento de Laboratorio

Aplicación web local para consultar equipos de laboratorio, comprobar su disponibilidad y gestionar reservas sin perder el historial de cancelaciones. El proyecto es un **MVP académico y funcional** desarrollado para demostrar un flujo de reservas, una arquitectura modular y reglas de negocio básicas.

## Objetivo y problema

El sistema busca ayudar a organizar el uso de equipamiento compartido. Sin un registro común, las reservas pueden superponerse y el historial de cancelaciones puede perderse. La aplicación concentra el inventario y las reservas en una base SQLite local y ofrece acceso desde un navegador.

## Alcance funcional

El MVP implementa:

- Consulta del inventario inicial y de los equipos agregados.
- Incorporación de equipos con identificador único y nombre.
- Consulta de disponibilidad por equipo e intervalo.
- Creación, consulta y cancelación de reservas.
- Conservación de reservas canceladas en el historial.
- Persistencia en SQLite entre ejecuciones.
- Validación de intervalos, existencia de equipos y superposiciones activas.
- Permiso para reservas consecutivas y reservas simultáneas de equipos distintos.
- Horizonte de reserva hasta seis meses calendario desde la fecha actual del servidor, inclusive.
- Validación de nombres de solicitantes: letras Unicode y espacios; se rechazan números y símbolos.
- Interfaz web local con autocompletado del final una hora después del inicio, que puede editarse.

El inventario precargado es MIC-01 y MIC-02 (microscopios digitales), OSC-01 (osciloscopio), MUL-01 (multímetro) y FUE-01 (fuente de alimentación).

La cancelación cambia el estado y registra una fecha, no elimina la fila. La regla de superposición solo toma en cuenta reservas `ACTIVE` del mismo equipo.

## Tecnologías

| Área | Tecnología |
|---|---|
| Lenguaje y versión del módulo | Go 1.27.1 |
| Servidor HTTP y API | `net/http` de Go, API REST con JSON |
| Persistencia | SQLite mediante `database/sql` y `modernc.org/sqlite` v1.61.0 |
| Interfaz | HTML5, CSS3 y JavaScript nativo con Fetch API |
| Pruebas de negocio e integración | Paquete estándar `testing` y SQLite real en bases temporales |

La aplicación no requiere servicios externos para funcionar localmente.

## Arquitectura

El sistema es un monolito modular. El frontend llama a la API REST; los handlers traducen las solicitudes HTTP a llamadas a servicios; los servicios aplican las reglas del dominio; y los repositorios consultan o modifican SQLite. Los handlers no acceden directamente a la base de datos.

| Ruta | Responsabilidad |
|---|---|
| `cmd/server` | Inicialización y arranque del servidor. |
| `internal/api` | Rutas REST, decodificación JSON y respuestas HTTP. |
| `internal/reservation` | Modelos, validaciones y reglas de reservas; repositorio SQLite. |
| `internal/inventory` | Validación, consulta y registro de equipos. |
| `internal/storage` | Conexión, esquema, migración compatible y carga del inventario inicial. |
| `web/` | Página, estilos y lógica de interacción del navegador. |

El flujo interno, la inicialización y los recorridos de las solicitudes se explican en [WORKFLOW.md](WORKFLOW.md). Las decisiones de arquitectura también están en [docs/architecture.md](docs/architecture.md) y [docs/technical-decisions.md](docs/technical-decisions.md).

## API REST

El servidor ofrece estas rutas locales:

| Método | Ruta | Propósito / entrada principal |
|---|---|---|
| `GET` | `/api/equipment` | Lista de equipos. |
| `POST` | `/api/equipment` | Agrega equipo; JSON `{ "id": "...", "name": "..." }`. |
| `GET` | `/api/equipment/{id}/availability?starts_at=...&ends_at=...` | Consulta disponibilidad con fechas RFC3339. |
| `POST` | `/api/reservations` | Crea una reserva; JSON con `equipment_id`, `requester`, `starts_at` y `ends_at`. |
| `GET` | `/api/reservations` | Lista reservas activas y canceladas. |
| `GET` | `/api/reservations/{id}` | Devuelve una reserva. |
| `PATCH` | `/api/reservations/{id}/cancel` | Cancela una reserva activa y conserva su registro. |

Las respuestas exitosas siguen el formato `{ "data": ... }`; los errores usan `{ "error": { "code": "...", "message": "..." } }`. La API valida JSON, limita los cuerpos y evita enviar detalles internos de SQLite al cliente. Más detalles del flujo y las respuestas están en [WORKFLOW.md](WORKFLOW.md).

## Requisitos y ejecución

- Go 1.27.1, de acuerdo con `go.mod`.
- Ejecutar los comandos desde la raíz del repositorio, ya que la ruta de la base y los archivos web son relativas al directorio actual.

Iniciá el servidor:

```powershell
go run ./cmd/server
```

Abrí [http://localhost:8080](http://localhost:8080). El servidor escucha en `127.0.0.1:8080`, crea `data/` y `data/reservations.db` cuando no existen, y carga los cinco equipos iniciales sin reemplazar los datos existentes. La interfaz se sirve desde `web/templates/index.html`; los recursos estáticos se publican bajo `/static/`.

La base local es información persistente del proyecto. No la borres si querés conservar equipos y reservas.

## Pruebas y verificaciones

El detalle de los escenarios, resultados, versiones y procedimientos está en [docs/testing.md](docs/testing.md).

**Pruebas ejecutadas automáticamente:** las pruebas Go usan SQLite real en bases temporales y cubren creación, validaciones, nombres válidos e inválidos, superposiciones, intervalos consecutivos, cancelación y persistencia al cerrar y reabrir SQLite. En el último registro, `go test ./... -count=1` y `go test ./... -cover` finalizaron correctamente; la cobertura de `internal/reservation` fue 78,6 %. También se ejecutó `node --check web/static/js/app.js`; no hay una suite automatizada de JavaScript.

**Verificaciones manuales:** el usuario informó que creó dos reservas a nombre de dos personas, canceló ambas, reinició el servidor y comprobó que los registros y sus fechas de cancelación se conservaron. Codex no realizó esa verificación. La verificación del autocompletado de fecha en un navegador sigue pendiente; su procedimiento reproducible está en `docs/testing.md`.

## Estado del proyecto y limitaciones

Este repositorio es un **MVP funcional con fines académicos y de aprendizaje**, diseñado para ejecución local. Prioriza la simplicidad y la comprensión de las reglas de negocio; no fue diseñado ni validado para cargas de producción.

El proyecto sí implementa validaciones de entrada y de reglas de reserva, validación de solicitudes JSON, límite de tamaño del cuerpo y escucha en loopback. Estas medidas acotan el comportamiento de la aplicación, pero **no constituyen una estrategia integral de seguridad ni un endurecimiento suficiente para producción**.

Antes de un despliegue productivo quedarían fuera del alcance actual, entre otros:

- Autenticación y autorización de usuarios.
- Gestión de roles y permisos.
- Protección y endurecimiento integral de endpoints.
- Registro estructurado y monitoreo/observabilidad.
- Estrategias de respaldo y recuperación de la base de datos.
- Escalabilidad y pruebas de carga.
- Manejo de concurrencia bajo condiciones de producción.
- Configuración segura y gestión de secretos.
- Despliegue, actualización y mantenimiento productivos.

Estas son posibles líneas de evolución, no funcionalidades implementadas por este MVP. Para las limitaciones técnicas y decisiones vigentes, consultar también [docs/technical-decisions.md](docs/technical-decisions.md).
