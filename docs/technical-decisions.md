# Decisiones técnicas

## Confirmadas e implementadas

- **Lenguaje:** Go 1.27.1, versión disponible en el entorno.
- **Persistencia local:** SQLite mediante `database/sql` y el controlador `modernc.org/sqlite` v1.61.0, importado sin nombre para registrar el driver `sqlite`.
- **Ubicación de la base:** `data/reservations.db`; el directorio `data/` se crea al inicializar si hace falta.
- **Esquema inicial:** tablas `equipment` y `reservations`; se crean con `CREATE TABLE IF NOT EXISTS`.
- **Inventario inicial:** cinco unidades con identificadores estables, definidos en `internal/inventory` e insertados desde `internal/storage`.
- **Idempotencia:** identificador de equipo como clave primaria e inserción `ON CONFLICT (id) DO NOTHING`; reiniciar no duplica ni sobrescribe equipos existentes.
- **Integridad referencial:** la tabla de reservas referencia a `equipment`; el driver activa claves foráneas para cada conexión mediante DSN.
- **Arquitectura:** monolito modular con módulos separados para API, reservas, inventario y almacenamiento.
- **Interfaz prevista:** HTML, CSS y JavaScript.
- **Comunicación prevista:** API REST local.
- **Servidor HTTP previsto:** biblioteca estándar `net/http`, aún sin servidor funcional.
- **Control de versiones:** Git.

## Pendientes

- Definir los contratos y rutas de la API REST.
- Definir validaciones y reglas detalladas para los intervalos y estados de reserva.
- Definir cómo evolucionará el esquema (migraciones) a medida que cambien los requisitos.
- Definir las operaciones de reservas y cancelación; todavía no están implementadas.