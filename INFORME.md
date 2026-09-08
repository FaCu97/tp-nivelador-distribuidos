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
| `BET_COUNT` | 2 bytes | Cantidad de apuestas, como uint16 |
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
| `NUMBER` | 2 bytes | Número como `uint16`. |

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
