# Arquitectura

La aplicación se organizará como un monolito modular: un único programa local con módulos internos separados por responsabilidad. El backend se implementará en Go y se comunicará con el frontend mediante una API REST servida en `localhost:8080`.

## Componentes previstos

- `cmd/server`: inicialización del programa y, en una etapa posterior, arranque del servidor.
- `internal/api`: endpoints HTTP y adaptación de solicitudes y respuestas.
- `internal/reservation`: operaciones y reglas relacionadas con reservas.
- `internal/inventory`: consulta y gestión del equipamiento.
- `internal/storage`: persistencia local mediante SQLite.
- `web/templates`: plantilla HTML de la interfaz.
- `web/static`: estilos CSS y JavaScript del frontend.
- `data`: ubicación para los archivos de datos locales.

Los módulos de `internal` encapsularán sus responsabilidades y evitarán que la interfaz acceda directamente a la persistencia. Esta descripción es una guía inicial; los límites e interfaces entre módulos se definirán durante el desarrollo.