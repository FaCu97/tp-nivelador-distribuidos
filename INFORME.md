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
| `SEND_BET` | `0x01` | Cliente -> servidor | Envía una apuesta. |
| `END_BETS` | `0x02` | Cliente -> servidor | Indica que la agencia terminó de enviar sus apuestas. No tiene payload. |
| `WINNERS_LIST` | `0x03` | Servidor -> cliente | Devuelve los ganadores correspondientes a la agencia. |
| `ACK` | `0x0A` | Servidor -> cliente | Confirmación, reservado para el protocolo. | (aun no implementado)
| `ERROR` | `0x0B` | Servidor -> cliente | Notificación de error, reservado para el protocolo. | (aun no implementado)

## Serialización de una apuesta

El payload de `SEND_BET` contiene una apuesta serializada con el siguiente
formato. Los campos numéricos utilizan big-endian:

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

El payload de `WINNERS_LIST` permite transportar una cantidad variable de
apuestas usando este formato:

```text
WINNERS_COUNT (uint32, 4 bytes)
WINNER_SIZE   (uint32, 4 bytes)
WINNER_DATA   (variable)
WINNER_SIZE   (uint32, 4 bytes)
WINNER_DATA   (variable)
...
```

`WINNERS_COUNT` indica cuántas apuestas siguen. Cada ganador tiene su propio
tamaño para que el cliente pueda recorrer el payload sin depender de un tamaño
fijo. El cliente interpreta este formato únicamente cuando el opcode recibido
es `WINNERS_LIST`, deserializa cada apuesta y persiste sus campos en
`OUTPUT_FILE`.

## Flujo de mensajes

Para cada agencia, el intercambio es:

```text
Cliente -> Servidor: SEND_BET (una vez por apuesta)
Cliente -> Servidor: END_BETS
Servidor -> Cliente: WINNERS_LIST
```
