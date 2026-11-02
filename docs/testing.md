# Pruebas automatizadas y verificaciones manuales

## Resumen de la última ejecución


| Comprobación                                 | Resultado                                                                   |
| -------------------------------------------- | --------------------------------------------------------------------------- |
| `go test ./... -count=1`                     | Correcto, código de salida 0                                                |
| `go test ./... -cover`                       | Correcto, código de salida 0                                                |
| Cobertura de `internal/reservation`          | **78,6 %**                                                                  |
| `node --check web/static/js/app.js`          | Correcto                                                                    |
| Reinicio real del servidor                   | Informado como realizado manualmente por el usuario; no ejecutado por Codex |
| Verificación del autocompletado en navegador | Pendiente de realizar manualmente                                           |


## Objetivo y estrategia

Las pruebas automatizadas comprueban las reglas del servicio de reservas y la persistencia usando SQLite real.

- Archivo de prueba: `internal/reservation/service_integration_test.go`.
- El paquete externo `reservation_test` construye los servicios y repositorios productivos; no usa mocks para las operaciones SQLite.
- Cada caso inicializa una base aislada en `t.TempDir()` mediante `storage.InitializeAt`, que ejecuta el esquema real y carga el inventario inicial.
- No se utiliza ni modifica `data/reservations.db`.
- Las fechas de los casos se calculan para respetar el límite dinámico de seis meses.
- No se incorporaron frameworks ni dependencias de testing.

## Validación de nombres de solicitantes

La validación de negocio está en `reservation.Service.Create`. Acepta letras Unicode y espacios, incluidos caracteres acentuados y `ñ`; también permite marcas diacríticas combinantes cuando siguen a una letra. Rechaza números, símbolos y espacios sin letras. Los espacios extremos se recortan antes de almacenar el nombre.

El frontend elimina caracteres ajenos al conjunto permitido mientras se escribe y muestra un mensaje si los quitó. También valida el contenido antes de enviar la solicitud. Esta validación no reemplaza la del backend.

Casos de aceptación comprobados automáticamente:

- `Juan Pérez`
- `María José`
- `José Núñez`
- Forma Unicode descompuesta de una vocal acentuada seguida del apellido `Núñez`.

Casos de rechazo comprobados automáticamente:

- `Juan123`
- `Juan@Pérez`
- `12345`
- Nombre compuesto únicamente por símbolos.
- Campo compuesto únicamente por espacios.

Los casos válidos se crean correctamente; los inválidos devuelven `ErrInvalidInput` y no persisten una reserva. Se ejecutaron con los dos comandos Go indicados abajo y todos pasaron. La suite ahora comprende **25 escenarios**, contando las subpruebas.

## Autocompletado de fecha y hora

Al cambiar el inicio, el JavaScript calcula el final sumando una hora y expresa el resultado en hora local. La suma contempla el cambio de día, mes y año. El usuario puede editar el final después del autocompletado.

Si el final automático supera el máximo de seis meses, el campo queda inválido, se muestra un mensaje para corregir el horario o el inicio y el formulario impide enviar la reserva. Se mantienen la validación de inicio anterior al final y la validación autoritativa del servicio Go.

### Procedimiento manual reproducible

1. Iniciar el servidor con `go run ./cmd/server` y abrir `http://localhost:8080`.
2. Seleccionar un equipo y escribir un nombre válido.
3. Elegir un inicio dentro del horizonte permitido y observar que el final queda una hora después.
4. Para comprobar el cambio de fecha, elegir un inicio a las 23:30 en una fecha válida; el final esperado es a las 00:30 del día siguiente.
5. Si el calendario permite fechas cercanas al cambio de mes o año dentro del horizonte, repetir allí y comprobar que el final cambia también de mes o año cuando corresponde.
6. Cambiar manualmente la fecha u hora final y comprobar que el valor editado se conserva y que la validación de intervalo continúa funcionando.
7. Elegir un inicio a las 23:30 del último día permitido. El final automático debe quedar fuera del límite, aparecer un mensaje comprensible y no debe enviarse la reserva. Corregir el inicio y comprobar que se puede continuar.

**Resultado observado por Codex:** no se ejecutó una verificación en navegador; queda pendiente. La sintaxis de `web/static/js/app.js` sí se comprobó con `node --check`.

## Persistencia tras reiniciar el servidor

**Verificación manual informada por el usuario; no ejecutada por Codex:** el usuario creó dos reservas a nombre de dos personas, canceló ambas, detuvo y reinició el servidor, y comprobó que los dos registros y sus fechas de cancelación se conservaron. No se registraron identificadores ni horarios.

Esta verificación corresponde a un reinicio real de la aplicación. Las pruebas automatizadas, por separado, cierran y reabren el archivo SQLite temporal; no simulan el ciclo de vida del servidor HTTP.

## Versiones del entorno


| Componente    | Versión y evidencia                                                                               |
| ------------- | ------------------------------------------------------------------------------------------------- |
| Go            | `go1.27.1 windows/amd64`, observado con `go version`; sin cambios respecto del registro anterior. |
| Driver SQLite | `modernc.org/sqlite v1.61.0`, declarado en `go.mod`; sin cambios respecto del registro anterior.  |


## Comandos ejecutados y resultados observados

Los comandos Go siguientes se ejecutaron realmente desde la raíz del repositorio el **2026-11-01**.

### `go test ./... -count=1`

**Resultado observado:** código de salida 0.


| Paquete                | Resultado              |
| ---------------------- | ---------------------- |
| `cmd/manual-check`     | Sin archivos de prueba |
| `cmd/server`           | Sin archivos de prueba |
| `internal/api`         | Sin archivos de prueba |
| `internal/inventory`   | Sin archivos de prueba |
| `internal/reservation` | Correcto, 1,256 s      |
| `internal/storage`     | Sin archivos de prueba |


### `go test ./... -cover`

**Resultado observado:** código de salida 0.


| Paquete                | Cobertura reportada  |
| ---------------------- | --------------------: |
| `cmd/manual-check`     | 0,0 %                |
| `cmd/server`           | 0,0 %                |
| `internal/api`         | 0,0 %                |
| `internal/inventory`   | 0,0 %                |
| `internal/reservation` | **78,6 %** (1,186 s) |
| `internal/storage`     | 0,0 %                |


El resto de los paquetes no tiene pruebas que instrumenten su cobertura en esta ejecución. No se observaron fallos ni se necesitaron correcciones.

También se ejecutó `node --check web/static/js/app.js`, que terminó correctamente. No se ejecutó una suite de pruebas JavaScript.

## Pendientes y revisión humana

- Ampliar en futuras etapas la cobertura automatizada de API, inventario, almacenamiento y JavaScript.