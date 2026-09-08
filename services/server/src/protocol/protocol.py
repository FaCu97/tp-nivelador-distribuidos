from enum import IntEnum

from lottery.bet import Bet

BIRTHDATE_SIZE = 10
NAME_MAX_SIZE = 255
BYTE_SIZE = 1
BATCH_COUNT_SIZE = 2
DOCUMENT_SIZE = 4
NUMBER_SIZE = 2
MAX_DOCUMENT = 0xFFFFFFFF
MAX_NUMBER = 0xFFFF

class Opcode(IntEnum):
    SEND_BETS = 0x01
    END_BETS = 0x02
    WINNERS_LIST = 0x03
    ACK = 0x0A
    ERROR = 0x0B

def decode_bets(data: bytes, agency_id: int) -> list[Bet]:
    def read_bytes(size: int, offset: int) -> tuple[bytes, int]:
        end = offset + size
        if end > len(data):
            raise ValueError("payload de apuesta incompleto")
        return data[offset:end], end

    raw, offset = read_bytes(BATCH_COUNT_SIZE, 0)
    bets_count = int.from_bytes(raw, byteorder="big")
    bets = []

    for _ in range(bets_count):
        raw, offset = read_bytes(BYTE_SIZE, offset)
        first_name_length = raw[0]
        first_name_bytes, offset = read_bytes(first_name_length, offset)
        first_name = first_name_bytes.decode("utf-8")

        raw, offset = read_bytes(BYTE_SIZE, offset)
        last_name_length = raw[0]
        last_name_bytes, offset = read_bytes(last_name_length, offset)
        last_name = last_name_bytes.decode("utf-8")

        document_bytes, offset = read_bytes(DOCUMENT_SIZE, offset)
        birthdate_bytes, offset = read_bytes(BIRTHDATE_SIZE, offset)
        number_bytes, offset = read_bytes(NUMBER_SIZE, offset)

        bets.append(Bet(
            agency_id=agency_id,
            first_name=first_name,
            last_name=last_name,
            document=int.from_bytes(document_bytes, "big"),
            birthdate=birthdate_bytes.decode("utf-8"),
            number=int.from_bytes(number_bytes, "big"),
        ))

    if len(bets) != bets_count:
        raise ValueError("cantidad de apuestas incorrecta")
    if offset != len(data):
        raise ValueError("payload de apuestas contiene bytes adicionales")
    return bets


def marshal_bet(bet: Bet) -> bytes:
    first_name = bet.first_name.encode("utf-8")
    last_name = bet.last_name.encode("utf-8")
    birthdate = bet.birthdate.encode("utf-8")

    if len(first_name) > NAME_MAX_SIZE or len(last_name) > NAME_MAX_SIZE:
        raise ValueError("nombre o apellido demasiado largos")
    if len(birthdate) != BIRTHDATE_SIZE:
        raise ValueError("la fecha debe tener exactamente 10 bytes")
    if bet.document < 0 or bet.document > MAX_DOCUMENT:
        raise ValueError("el documento no entra en uint32")
    if bet.number < 0 or bet.number > MAX_NUMBER:
        raise ValueError("el número no entra en uint16")

    return (
        bytes([len(first_name)])
        + first_name
        + bytes([len(last_name)])
        + last_name
        + bet.document.to_bytes(4, byteorder="big")
        + birthdate
        + bet.number.to_bytes(2, byteorder="big")
    )

def marshal_bets(bets: list[Bet]) -> bytes:
    payloads = [marshal_bet(bet) for bet in bets]
    return (
        len(payloads).to_bytes(BATCH_COUNT_SIZE, byteorder="big")
        + b"".join(payload for payload in payloads)
    )
