package safe_socket

import "io"

//TODO: Complete with a short-read/short-write tolerant implementation

func SendAll(socket io.Writer, bytes []byte) error {
	bytesWritten := 0
	bytesToWrite := len(bytes)

	for bytesWritten < bytesToWrite {
		n, err := socket.Write(bytes[bytesWritten:])
		if err != nil {
			return err
		}
		bytesWritten += n
	}
	return nil
}

func RecvAll(socket io.Reader, size int) ([]byte, error) {
	bytesRead := 0
	bytesToRead := size

	buff := make([]byte, size)
	for bytesRead < bytesToRead {

		n, err := socket.Read(buff[bytesRead:])
		bytesRead += n
		if err != nil {
			return buff[:bytesRead], err
		}
	}
	return buff, nil
}
