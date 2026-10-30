# Requerimientos (borrador inicial)

Este documento describe el alcance previsto. Los detalles de validación y las reglas específicas pendientes deben acordarse antes de implementar cada módulo.

## Requerimientos funcionales

- RF-01: consultar el equipamiento de laboratorio y su disponibilidad.
- RF-02: registrar una reserva para un equipo.
- RF-03: cancelar una reserva existente.
- RF-04: conservar un historial de operaciones de reserva y cancelación.

## Requerimientos no funcionales

- RNF-01: funcionar localmente y ser accesible desde un navegador en `localhost:8080`.
- RNF-02: operar sin depender de Internet ni de servicios externos.
- RNF-03: organizar el código como un monolito modular con responsabilidades separadas.
- RNF-04: utilizar Go para el backend y SQLite como persistencia prevista.
- RNF-05: exponer la comunicación entre frontend y backend mediante una API REST local.

## Reglas de negocio iniciales

- RN-01: las reservas deben corresponder a equipamiento del inventario.
- RN-02: la disponibilidad debe considerar las reservas existentes.
- RN-03: cancelar una reserva no debe eliminar su registro histórico.

Los intervalos, estados, permisos y validaciones detalladas quedan pendientes de definición.