# Arquitectura

La aplicación se organiza como un monolito modular: un único programa local con módulos internos separados por responsabilidad. El backend está escrito en Go; la inicialización actual prepara la persistencia local. La API REST y la interfaz web quedan para etapas posteriores.

## Componentes

- `cmd/server`: punto de entrada. Inicializa SQLite al arrancar el programa.
- `internal/storage`: crea el directorio y el archivo SQLite, prepara el esquema e inserta el catálogo inicial de forma idempotente.
- `internal/inventory`: define el tipo `Equipment` y el catálogo inicial de equipamiento.
- `internal/reservation`: reservado para operaciones y reglas de reservas futuras; no implementa creación ni cancelación.
- `internal/api`: reservado para la futura API HTTP; todavía no contiene endpoints.
- `web/templates` y `web/static`: estructura inicial de la futura interfaz.
- `data`: almacena el archivo local `reservations.db`, creado automáticamente al iniciar el programa.

El módulo de almacenamiento depende del catálogo de inventario para cargar los equipos. La lógica de reservas no se mezcla con la inicialización ni el acceso a la base de datos.