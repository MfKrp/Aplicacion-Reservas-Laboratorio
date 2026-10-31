# Decisiones técnicas

## Confirmadas e implementadas

- **Lenguaje:** Go 1.27.1.
- **Persistencia:** SQLite mediante `database/sql` y `modernc.org/sqlite` v1.61.0.
- **Ubicación:** `data/reservations.db`; el directorio local se crea al inicializar.
- **Arquitectura:** monolito modular con paquetes para dominio, reservas, inventario y almacenamiento.
- **Estados:** los valores persistidos por el repositorio son `ACTIVE` y `CANCELLED`.
- **Fechas de entrada:** RFC3339 con zona horaria explícita. Se normalizan a UTC y se persisten con nueve posiciones de fracción de segundo para comparar intervalos de forma ordenable como texto.
- **Solapamiento:** condición estricta `starts_at < new_end AND ends_at > new_start`, filtrada por equipo y estado `ACTIVE`. Los extremos iguales permiten reservas consecutivas.
- **Atomicidad del alta:** el repositorio verifica solapamiento e inserta en la misma transacción. La conexión SQLite está configurada con un pool de una conexión para este uso local.
- **Cancelación:** una actualización condicional conserva la fila, cambia su estado a `CANCELLED` y guarda `canceled_at`; la consulta de disponibilidad ignora estados cancelados.
- **Esquema:** se añadió `requester` a `reservations`, campo necesario para persistir el solicitante requerido. La inicialización inspecciona `PRAGMA table_info` y ejecuta `ALTER TABLE ... ADD COLUMN` solo si falta, con `DEFAULT ''` para conservar filas preexistentes. Las altas nuevas siempre envían un solicitante validado.
- **Consulta de inventario:** `internal/inventory` consulta por ID; la entidad `Equipment` está definida una sola vez en el dominio y se comparte mediante alias.

## Pendientes de revisión o definición

- Validar con el usuario el formato RFC3339 y la precisión de nanosegundos elegida para la persistencia.
- Confirmar los campos del esquema `reservations` y si se necesitan migraciones versionadas a futuro.
- La regla de intervalos no prohíbe fechas pasadas porque el requerimiento no lo especifica.
- Los estados son constantes de aplicación, pero el esquema SQLite existente no impone un `CHECK` para limitar otros valores. Además, el default histórico de `status` es `active` en minúsculas; el repositorio evita depender de él y siempre escribe `ACTIVE` explícitamente. Revisar si se normalizará ese default en una migración futura.
- Definir contratos y endpoints de API REST; no están implementados.
- Definir autenticación y permisos solo si se incorporan como requisito posterior.