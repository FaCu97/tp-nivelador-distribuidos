package safe_socket

import (
    "encoding/binary"
    "io"
)

const HEADER_SIZE = 6

const (
	OpSendBets     uint8 = 0x01
	OpEndBets     uint8 = 0x02
	OpWinnersList uint8 = 0x03
	OpAck         uint8 = 0x0A
	OpError       uint8 = 0x0B
)

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

func SendFrame(socket io.Writer, opcode byte, agency byte, payload []byte) error {
	header := make([]byte, HEADER_SIZE)
	header[0] = opcode
	header[1] = agency

	payloadSize := uint32(len(payload))
	binary.BigEndian.PutUint32(header[2:], payloadSize)

	if err := SendAll(socket, header); err != nil {
		return err
	}
	if len(payload) > 0 {
		return SendAll(socket, payload)
	}
	return nil
}

func RecvFrame(socket io.Reader) (byte, byte, []byte, error) {
	header, err := RecvAll(socket, HEADER_SIZE)
	if err != nil {
		return 0, 0, nil, err
	}

	opcode := header[0]
	agency := header[1]
	payloadSize := binary.BigEndian.Uint32(header[2:])
	payload, err := RecvAll(socket, int(payloadSize))
	if err != nil {
		return 0, 0, nil, err
	}

	return opcode, agency, payload, nil
}