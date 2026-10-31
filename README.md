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

- Inicialización de SQLite en `data/reservations.db`, creación del esquema e inventario inicial.
- Dominio de reservas con estados `ACTIVE` y `CANCELLED`.
- Repositorio SQLite para crear, listar, consultar y cancelar reservas.
- Servicio con validaciones, consulta de disponibilidad y prevención de superposiciones activas.
- Conservación del registro y fecha al cancelar una reserva.
- Verificador manual directo del servicio en `cmd/manual-check`, con base independiente `data/manual-check.db` que se limpia en cada ejecución.
- API REST local en `internal/api`, con seis endpoints JSON y respuestas de error centralizadas.

**Pendiente:**

- Interfaz web conectada al backend.
- Pruebas automatizadas, previstas para una etapa posterior.

La API se inicia desde PowerShell, en la raíz del repositorio:

```powershell
go run ./cmd/server
```

El verificador manual del dominio sigue disponible con `go run ./cmd/manual-check`.

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
