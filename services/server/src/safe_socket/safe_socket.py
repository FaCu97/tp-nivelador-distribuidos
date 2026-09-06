import socket

# TODO: Complete with a short-read/short-write tolerant implementation


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
