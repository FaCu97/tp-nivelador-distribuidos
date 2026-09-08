import socket
import struct

HEADER_SIZE = 6



def recv_all(socket: socket.socket, size):
    bytesToRead = size
    bytesRead = 0
    chunks = []
    while bytesRead < bytesToRead:
        chunk = socket.recv(bytesToRead - bytesRead) 
        bytesRead += len(chunk)
        chunks.append(chunk)
        if chunk == b"":
            raise RuntimeError("socket connection broken")

    return b"".join(chunks)


def send_all(socket: socket.socket, bytes):
    bytesToWrite = len(bytes)
    bytesWritten = 0
    while bytesWritten < bytesToWrite:
        n = socket.send(bytes[bytesWritten:])
        bytesWritten += n
        if n == 0:
            raise RuntimeError("socket connection broken")
        
    return bytesWritten


def recv_frame(socket: socket.socket):
    header = recv_all(socket, HEADER_SIZE)
    opcode = header[0]
    agency = header[1]
    payload_size = struct.unpack(">I", header[2:])[0]
    payload = recv_all(socket, payload_size)
    return opcode, agency, payload


def send_frame(socket: socket.socket, opcode: int, agency: int, payload: bytes):
    header = bytes([opcode, agency]) + struct.pack(">I", len(payload))
    return send_all(socket, header + payload)


