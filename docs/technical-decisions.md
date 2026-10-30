# Decisiones técnicas

## Confirmadas

- **Lenguaje:** Go. El entorno dispone de Go 1.27.1.
- **Interfaz:** HTML, CSS y JavaScript.
- **Comunicación:** API REST entre la interfaz y el backend.
- **Ejecución:** aplicación local accesible mediante `localhost:8080`, sin servicios externos.
- **Arquitectura:** monolito modular con separación de responsabilidades.
- **Control de versiones:** Git.

## Previstas, pendientes de implementación

- **Persistencia:** SQLite. El esquema y la estrategia de acceso todavía no están definidos.
- **Servidor HTTP:** se prevé utilizar la biblioteca estándar `net/http`; todavía no hay servidor funcional.

## Pendientes

- Selección del driver SQLite. No se seleccionó ni se agregó ninguno.
- Esquema de base de datos y política de migraciones.
- Contratos y formato de los endpoints REST.
- Reglas de validación detalladas para reservas y disponibilidad.