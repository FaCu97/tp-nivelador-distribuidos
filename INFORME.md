# Protocolo de comunicación

La comunicación entre clientes y servidor se realiza mediante sockets TCP. Para
separar mensajes dentro del flujo de bytes se utiliza un frame con un header de
tamaño fijo y un payload de tamaño variable.

## Header

Todos los frames tienen el siguiente header de 6 bytes:

| Campo | Tamaño | Descripción |
|---|---:|---|
| `OPCODE` | 1 byte | Tipo de mensaje. |
| `AGENCY_ID` | 1 byte | Identificador de la agencia que origina el mensaje. |
| `PAYLOAD_SIZE` | 4 bytes | Tamaño del payload, representado en big-endian. |

El payload comienza inmediatamente después del header.

## Opcodes

| Opcode | Valor | Dirección | Descripción |
|---|---:|---|---|
| `SEND_BETS` | `0x01` | Cliente -> servidor | Envía un batch de apuestas. |
| `END_BETS` | `0x02` | Cliente -> servidor | Indica que la agencia terminó de enviar sus apuestas. No tiene payload. |
| `WINNERS_LIST` | `0x03` | Servidor -> cliente | Devuelve los ganadores correspondientes a la agencia. |
| `ACK` | `0x0A` | Servidor -> cliente | Confirma que el batch fue procesado correctamente. |
| `ERROR` | `0x0B` | Servidor -> cliente | Notificación de error, reservado para el protocolo. |

## Serialización de una apuesta

El payload de `SEND_BETS` contiene un batch de apuestas. Los campos numéricos
utilizan big-endian:

| Campo | Tamaño | Descripción |
|---|---:|---|
| `BET_COUNT` | 2 bytes | Cantidad de apuestas del batch como `uint16`. |
| `BET_1 ... BET_N` | Variable | Apuestas serializadas consecutivamente. |

Cada apuesta tiene este formato:

| Campo | Tamaño | Descripción |
|---|---:|---|
| `FIRST_NAME_SIZE` | 1 byte | Cantidad de bytes del nombre. |
| `FIRST_NAME` | Variable | Nombre codificado en UTF-8. |
| `LAST_NAME_SIZE` | 1 byte | Cantidad de bytes del apellido. |
| `LAST_NAME` | Variable | Apellido codificado en UTF-8. |
| `DOCUMENT` | 4 bytes | Documento como `uint32`. |
| `BIRTHDATE` | 10 bytes | Fecha en formato `YYYY-MM-DD`. |
| `NUMBER` | 4 bytes | Número como `uint32`. |

## Lista de ganadores

El payload de `WINNERS_LIST` utiliza el mismo formato de lista:

```text
BET_COUNT (uint16, 2 bytes)
BET_1
BET_2
...
```

El cliente interpreta este payload cuando recibe el opcode `WINNERS_LIST` y
persiste los ganadores correspondientes a su agencia en `OUTPUT_FILE`.

El campo `NUMBER` ocupa 4 bytes y se representa como `uint32` en big-endian,
tanto en los batches enviados por el cliente como en la lista de ganadores.

## Concurrencia y sincronización

El servidor mantiene un hilo principal que crea el socket TCP, ejecuta
`bind` y `listen`, y permanece aceptando conexiones. Cada conexión aceptada
se procesa en un hilo de agencia independiente. De esta forma, una agencia
puede continuar enviando sus batches mientras el hilo principal acepta nuevas
conexiones y otros hilos atienden a las demás agencias.

Todas las agencias comparten la instancia de `Lottery`, el cual está protegido con
`file_lock`. El lock cubre solamente la escritura del batch y se libera antes
de enviar el `ACK`, evitando mantenerlo durante operaciones de red.

Al recibir `END_BETS`, el hilo de la agencia informa que terminó su ingesta.
Para ello incrementa `agencias_listas` dentro de `quorum_cond`, que también
protege el acceso al contador, y ejecuta `notify`. Un hilo de sorteo único
espera con `wait_for` hasta que `agencias_listas` alcance
`AGENCY_QUORUM_MIN`.

Una vez alcanzado el quórum, el hilo de sorteo lee las apuestas, calcula los
ganadores y construye `ganadores_memoria`, agrupando los resultados por
`agency_id`. Luego activa `sorteo_listo`, un `Event` compartido que libera a
los hilos de agencia que estaban esperando. Cada hilo consulta únicamente la
entrada correspondiente a su propia agencia, serializa esa lista y envía su
`WINNERS_LIST`


## Flujo de mensajes

Para cada agencia, el intercambio es:

```text
Cliente -> Servidor: SEND_BETS (batch 1)
Servidor -> Cliente: ACK
Cliente -> Servidor: SEND_BETS (batch 2)
Servidor -> Cliente: ACK
...
Cliente -> Servidor: END_BETS
Servidor -> Cliente: WINNERS_LIST
```

El cliente espera el `ACK` de cada batch antes de enviar el siguiente. El
servidor envía el `ACK` únicamente después de deserializar y almacenar todas
las apuestas del batch. Si el payload es inválido o no coincide con la
cantidad declarada, no se confirma el batch.

## Terminación Graceful (SIGTERM)

- En el cliente: Se captura la señal del sistema operativo mediante signal.Notify() en un hilo independiente. Al recibirla, se activa un canal shutdown que funciona como semáforo. El ciclo de lectura y envío principal (client.go) consulta constantemente el método client.IsShutdown(); si este retorna verdadero, el cliente aborta la ejecución limpiamente y retorna antes de que docker lance un SIGKILL.

- En el servidor: Se asocia un manejador de señales con signal.signal(). Al recibir la señal, se setea self.shutdown_event.set() y se notifica a todos los hilos. Esto destraba el hilo del lottery y los hilos de los clientes conectados, los cuales evalúan la condición del evento en sus ciclos while, saliendo de sus bucles y cerrando de forma segura sus sockets de comunicación.