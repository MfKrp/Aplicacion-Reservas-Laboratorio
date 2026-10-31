# Sistema de Reservas de Equipamiento de Laboratorio

Sistema local de reservas de equipamiento de laboratorio en Go y SQLite. Cuenta con persistencia, lógica de reservas y una API REST local; la interfaz web todavía no está conectada.

**Estado del proyecto:** Dominio, persistencia SQLite, API REST y verificador manual implementados. Interfaz web pendiente.

## 1. Especificación

### 1.1. Descripción del problema

La coordinación de un laboratorio necesita gestionar las reservas de sus equipos de manera organizada, evitando conflictos de horarios y conservando el historial de operaciones.

El sistema permitirá consultar la disponibilidad de cada unidad física, registrar nuevas reservas y cancelar reservas existentes sin perder sus datos históricos.

### 1.2. Objetivo

Desarrollar una aplicación web local que permita administrar las reservas de equipamiento de laboratorio, garantizando la integridad de los intervalos reservados y la persistencia de la información.

### 1.3. Alcance funcional

El sistema deberá permitir:

- Consultar el inventario precargado.
- Consultar la disponibilidad de una unidad durante un intervalo determinado.
- Crear reservas indicando equipo, solicitante, fecha y horario.
- Asignar un identificador único a cada reserva.
- Validar los datos ingresados.
- Rechazar reservas que se superpongan con otras reservas activas de la misma unidad.
- Permitir reservas consecutivas y reservas simultáneas de unidades diferentes.
- Cancelar reservas sin eliminar sus registros.
- Liberar los intervalos correspondientes a las reservas canceladas.
- Persistir la información entre ejecuciones de la aplicación.

### 1.4. Inventario inicial

| Identificador | Equipo |
|---|---|
| MIC-01 | Microscopio digital |
| MIC-02 | Microscopio digital |
| OSC-01 | Osciloscopio |
| MUL-01 | Multímetro |
| FUE-01 | Fuente de alimentación |

Cada identificador representa una unidad física independiente.

### 1.5. Reglas de negocio

- No se pueden reservar unidades inexistentes.
- El solicitante no puede estar vacío ni contener únicamente espacios.
- La fecha y hora de inicio deben ser anteriores a las de finalización.
- Una unidad no puede tener dos reservas activas con intervalos superpuestos.
- Se permiten reservas consecutivas.
- Las unidades diferentes pueden reservarse simultáneamente.
- Las reservas canceladas no bloquean la disponibilidad.
- La cancelación conserva el identificador y los datos originales de la reserva.

### 1.6. Fuera de alcance

En la primera versión no se contempla:

- Autenticación y gestión de usuarios reales.
- Administración del inventario.
- Notificaciones.
- Pagos.
- Integraciones con servicios externos.
- Interfaz gráfica compleja.
- Despliegue en la nube.
- Concurrencia distribuida.
- Escalabilidad de producción.

## 2. Solución técnica

### 2.1. Arquitectura

Se utilizará una arquitectura de monolito modular con separación de responsabilidades.

La aplicación estará compuesta por:

- Frontend web: HTML, CSS y JavaScript.
- Backend: Go.
- Servidor HTTP: paquete estándar `net/http`.
- Comunicación: API REST utilizando JSON.
- Persistencia: SQLite.
- Acceso a datos: paquete `database/sql` con el driver `modernc.org/sqlite`.

### 2.2. Funcionamiento

El backend se ejecuta localmente y expone la API en loopback:

`http://localhost:8080`

El servidor HTTP ya está implementado y escucha únicamente en `127.0.0.1:8080`. La interfaz web todavía no está conectada. El proyecto no depende de servicios externos en ejecución.

### 2.3. Organización del proyecto

```text
reservation-system/
│
├── cmd/
│   ├── server/main.go
│   └── manual-check/main.go
├── internal/
│   ├── api/handler.go
│   ├── reservation/
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── inventory/
│   │   ├── equipment.go
│   │   └── repository.go
│   └── storage/database.go
├── web/
│   ├── templates/
│   └── static/
│       ├── css/
│       └── js/
├── data/
├── docs/
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

### 2.4. Estado de implementación

**Implementado:**

- Inicialización de SQLite en `data/reservations.db` e inventario inicial.
- Dominio y servicio de reservas con validación, consulta de disponibilidad, prevención de superposiciones y cancelación con historial.
- Verificador manual del dominio en `cmd/manual-check`, que recrea una base de datos independiente en cada ejecución.
- API REST local con `net/http`, dependencias inyectadas y respuestas JSON centralizadas.

**Pendiente:**

- Interfaz web conectada al backend.
- Pruebas automatizadas, previstas para una etapa posterior.

### 2.5. API REST

El servidor escucha solo en `127.0.0.1:8080` y ofrece estas rutas:

| Método | Ruta | Uso |
|---|---|---|
| GET | `/api/equipment` | Listar equipos. |
| GET | `/api/equipment/{id}/availability?starts_at=...&ends_at=...` | Consultar disponibilidad para un intervalo RFC3339. |
| POST | `/api/reservations` | Crear una reserva. |
| GET | `/api/reservations` | Listar reservas activas y canceladas. |
| GET | `/api/reservations/{id}` | Consultar una reserva por ID. |
| PATCH | `/api/reservations/{id}/cancel` | Cancelar una reserva conservando el historial. |

El POST recibe `Content-Type: application/json` y un único objeto. Se rechazan campos desconocidos y cuerpos mayores a 1 MiB. Las respuestas exitosas usan `{"data": ...}`; los errores usan `{"error":{"code":"...","message":"..."}}`.

Códigos principales: **200** para consultas y cancelación, **201** al crear, **400** para solicitudes inválidas, **404** para recursos inexistentes, **405** para métodos no permitidos, **409** para conflictos de reserva, **413** para cuerpos demasiado grandes, **415** para tipo de contenido incorrecto y **500** para errores internos sin detalles de SQLite.

Ejemplo de solicitud para crear una reserva:

```json
{
  "equipment_id": "MIC-01",
  "requester": "Ana Pérez",
  "starts_at": "2030-06-15T09:00:00-03:00",
  "ends_at": "2030-06-15T11:00:00-03:00"
}
```

### 2.6. Ejecución y verificación manual

En una terminal de PowerShell, desde la raíz del repositorio, iniciar el servidor:

```powershell
go run ./cmd/server
```

En otra terminal, consultar el inventario y la disponibilidad, crear una reserva y luego consultarla y cancelarla:

```powershell
$base = "http://127.0.0.1:8080"
Invoke-RestMethod "$base/api/equipment"

$starts = [uri]::EscapeDataString("2030-06-15T09:00:00-03:00")
$ends = [uri]::EscapeDataString("2030-06-15T11:00:00-03:00")
Invoke-RestMethod "$base/api/equipment/MIC-01/availability?starts_at=$starts&ends_at=$ends"

$body = @{
  equipment_id = "MIC-01"
  requester = "Ana Pérez"
  starts_at = "2030-06-15T09:00:00-03:00"
  ends_at = "2030-06-15T11:00:00-03:00"
} | ConvertTo-Json

$created = Invoke-RestMethod -Method Post `
  -Uri "$base/api/reservations" `
  -ContentType "application/json" `
  -Body $body
$id = $created.data.id

Invoke-RestMethod "$base/api/reservations"
Invoke-RestMethod "$base/api/reservations/$id"
Invoke-RestMethod -Method Patch -Uri "$base/api/reservations/$id/cancel"
```

El verificador del dominio se ejecuta por separado con `go run ./cmd/manual-check` y utiliza `data/manual-check.db`, no la base principal.
## 3. Preparación y utilización del agente

El desarrollo contará con un agente de programación que trabajará directamente sobre el repositorio.

Se utilizará un entorno existente de asistencia a la programación, con acceso controlado a los archivos del proyecto.

### 3.1. Criterios de utilización

- Proporcionar instrucciones específicas para cada tarea.
- Delegar funcionalidades acotadas y verificables.
- Mantener las decisiones arquitectónicas documentadas.
- Revisar los cambios generados antes de incorporarlos.
- Ejecutar las verificaciones correspondientes a medida que avance el desarrollo.
- Conservar evidencia de las interacciones y resultados relevantes.

### 3.2. Registro del agente

Los datos reales del entorno, modelo, versión, permisos e instrucciones utilizadas se documentarán a medida que se realicen las delegaciones.

No se considerarán realizadas aquellas operaciones que todavía no hayan sido ejecutadas y verificadas.

## 4. Registro de decisiones y delegación

Esta sección conservará la evolución del proyecto y las decisiones tomadas durante el desarrollo.

### 4.1. Decisiones iniciales

| Decisión | Elección | Motivo |
|---|---|---|
| Lenguaje | Go | Desarrollo del backend y aprendizaje del lenguaje. |
| Arquitectura | Monolito modular | Simplicidad y separación de responsabilidades. |
| Interfaz | Web local | Acceso mediante navegador sin dependencia de Internet. |
| Base de datos | SQLite con `modernc.org/sqlite` | Persistencia local sin servidor independiente; driver sin CGO. |
| API | `net/http` estándar | Menos dependencias y escucha restringida a loopback. |
| Comunicación | API REST | Separación entre frontend y backend. |
| Frontend | HTML, CSS y JavaScript | Evitar frameworks innecesarios inicialmente. |

### 4.2. Episodios de delegación

Las delegaciones y sus resultados verificables se registran en `docs/agent-work-log.md`; la revisión humana queda indicada en cada ficha.

Cada episodio incluirá:

1. Objetivo y requisito.
2. Contexto e instrucciones suministradas.
3. Acción o resultado generado por el agente.
4. Revisión y decisión humana.
5. Evidencia verificable vinculada al repositorio.

No se registrarán episodios ficticios ni resultados que no hayan sido comprobados.

### 4.3. Evolución del proyecto

Las decisiones y modificaciones significativas se conservarán mediante Git.

Los commits deberán representar cambios coherentes, identificables y revisables.

---

## Estado actual

El repositorio incluye persistencia SQLite, inventario, dominio de reservas, API REST local y un verificador manual con base independiente.

La interfaz web y las pruebas automatizadas siguen pendientes.
