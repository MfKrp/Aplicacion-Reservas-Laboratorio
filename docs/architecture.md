# Arquitectura

La aplicación es un monolito modular Go. Los módulos separan modelos de dominio, reglas de reservas, consultas de inventario y persistencia SQLite. La futura API HTTP y la interfaz todavía no están implementadas.

## Componentes

- `cmd/server`: inicializa la base local; la composición del servicio con sus repositorios queda disponible para la siguiente integración.
- `internal/reservation/model.go`: modelos `Equipment` y `Reservation`, estados y errores del dominio.
- `internal/reservation/service.go`: validación de entradas y reglas de creación, consulta, disponibilidad y cancelación.
- `internal/reservation/repository.go`: interfaz de persistencia y operaciones SQL de reservas. El alta comprueba superposiciones e inserta dentro de una transacción.
- `internal/inventory/`: catálogo inicial y consultas SQL de equipos. El módulo expone el modelo `Equipment` del dominio.
- `internal/storage/`: conexión y configuración de SQLite, esquema inicial y migración compatible de columnas.
- `internal/api/`: reservado para endpoints HTTP futuros.
- `web/`: reservado para la interfaz HTML, CSS y JavaScript.
- `data/`: archivo SQLite local `reservations.db`.

El servicio depende de interfaces de consulta y persistencia; no conoce detalles SQL. El repositorio no sustituye las validaciones del servicio, pero realiza en una sola transacción la comprobación de solapamiento y el insert para evitar que dos altas secuenciales dentro del proceso ocupen el mismo intervalo.