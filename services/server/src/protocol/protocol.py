from enum import IntEnum

from lottery.bet import Bet

BIRTHDATE_SIZE = 10
NAME_MAX_SIZE = 255

class Opcode(IntEnum):
    SEND_BET = 0x01
    END_BETS = 0x02
    WINNERS_LIST = 0x03
    ACK = 0x0A
    ERROR = 0x0B

def decode_bet(data: bytes, agency_id: int) -> Bet:
    offset = 0

    def read_bytes(size: int, offset: int) -> tuple[bytes, int]:
        end = offset + size
        if end > len(data):
            raise ValueError("payload de apuesta incompleto")
        return data[offset:end], end

    raw, offset = read_bytes(1, offset)
    first_name = data[offset:offset + raw[0]].decode("utf-8")
    offset += raw[0]

    raw, offset = read_bytes(1, offset)
    last_name = data[offset:offset + raw[0]].decode("utf-8")
    offset += raw[0]

    document_bytes, offset = read_bytes(4, offset)
    document = int.from_bytes(document_bytes, "big")

    birthdate_bytes, offset = read_bytes(10, offset)
    birthdate = birthdate_bytes.decode("utf-8")

    number_bytes, offset = read_bytes(2, offset)
    number = int.from_bytes(number_bytes, "big")

    if offset != len(data):
        raise ValueError("payload de apuesta contiene bytes adicionales")

    return Bet(
        agency_id=agency_id,
        first_name=first_name,
        last_name=last_name,
        document=document,
        birthdate=birthdate,
        number=number,
    )


def marshal_bet(bet: Bet) -> bytes:
    first_name = bet.first_name.encode("utf-8")
    last_name = bet.last_name.encode("utf-8")
    birthdate = bet.birthdate.encode("utf-8")

    if len(first_name) > NAME_MAX_SIZE or len(last_name) > NAME_MAX_SIZE:
        raise ValueError("nombre o apellido demasiado largos")
    if len(birthdate) != BIRTHDATE_SIZE:
        raise ValueError("la fecha debe tener exactamente 10 bytes")
    if bet.document < 0 or bet.document > 0xFFFFFFFF:
        raise ValueError("el documento no entra en uint32")
    if bet.number < 0 or bet.number > 0xFFFF:
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
        len(payloads).to_bytes(4, byteorder="big")
        + b"".join(payload for payload in payloads)
    )
