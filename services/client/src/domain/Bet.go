package domain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type Bet struct {
	Name   string
	LastName string
	Dni	uint32
	Date   string
    Number uint32
}

const (
	maxNameSize    = 255
	birthdateSize  = 10
	batchCountSize = 2
	maxBatchCount  = 1<<16 - 1
)

func (bet *Bet) MarshalInto(buf *bytes.Buffer) error {
	nameBytes := []byte(bet.Name)
	lastNameBytes := []byte(bet.LastName)

	if len(nameBytes) > maxNameSize || len(lastNameBytes) > maxNameSize {
		return fmt.Errorf("nombre o apellido demasiado largos (máximo 255 bytes)")
	}

	// 1. NAME_SIZE (uint8) + NAME
	if err := buf.WriteByte(byte(len(nameBytes))); err != nil {
		return err
	}
	if _, err := buf.Write(nameBytes); err != nil {
		return err
	}

	// 2. LAST_NAME_SIZE (uint8) + LAST_NAME
	if err := buf.WriteByte(byte(len(lastNameBytes))); err != nil {
		return err
	}
	if _, err := buf.Write(lastNameBytes); err != nil {
		return err
	}

	// 3. DNI (uint32) -> 4 bytes en Big-Endian
	if err := binary.Write(buf, binary.BigEndian, bet.Dni); err != nil {
		return err
	}

	// 4. DATE (string 10 bytes fijos)
	dateBytes := []byte(bet.Date)
	if len(dateBytes) != birthdateSize {
		return fmt.Errorf("la fecha debe medir exactamente 10 bytes (YYYY-MM-DD), midió %d", len(dateBytes))
	}
	buf.Write(dateBytes)

	// 5. NUMBER (uint32) -> 4 bytes en Big-Endian (Network Byte Order)
	if err := binary.Write(buf, binary.BigEndian, bet.Number); err != nil {
		return err
	}

	return nil
}

func MarshalBets(bets []*Bet, buf *bytes.Buffer) error {
	buf.Reset()
	if len(bets) > maxBatchCount {
		return fmt.Errorf("demasiadas apuestas en el batch")
	}
	if err := binary.Write(buf, binary.BigEndian, uint16(len(bets))); err != nil {
		return err
	}
	for _, bet := range bets {
		if err := bet.MarshalInto(buf); err != nil {
			return err
		}
	}
	return nil
}

func UnmarshalBet(data []byte) (*Bet, int, error) {
	buf := bytes.NewReader(data)
	bet := &Bet{}

	// 1. NAME_SIZE (uint8) + NAME
	var nameSize uint8
	if err := binary.Read(buf, binary.BigEndian, &nameSize); err != nil {
		return nil, 0, err
	}
	nameBytes := make([]byte, nameSize)
	if bytesRead, err := buf.Read(nameBytes); err != nil {
		return nil, 0, err
	} else if bytesRead != len(nameBytes) {
		return nil, 0, fmt.Errorf("nombre incompleto")
	}
	bet.Name = string(nameBytes)

	// 2. LAST_NAME_SIZE (uint8) + LAST_NAME
	var lastNameSize uint8
	if err := binary.Read(buf, binary.BigEndian, &lastNameSize); err != nil {
		return nil, 0, err
	}
	lastNameBytes := make([]byte, lastNameSize)
	if bytesRead, err := buf.Read(lastNameBytes); err != nil {
		return nil, 0, err
	} else if bytesRead != len(lastNameBytes) {
		return nil, 0, fmt.Errorf("apellido incompleto")
	}
	bet.LastName = string(lastNameBytes)

	// 3. DNI (uint32) -> 4 bytes en Big-Endian
	if err := binary.Read(buf, binary.BigEndian, &bet.Dni); err != nil {
		return nil, 0, err
	}

	// 4. DATE (string 10 bytes fijos)
	dateBytes := make([]byte, birthdateSize)
	if bytesRead, err := buf.Read(dateBytes); err != nil {
		return nil, 0, err
	} else if bytesRead != len(dateBytes) {
		return nil, 0, fmt.Errorf("fecha incompleta")
	}
	bet.Date = string(dateBytes)

	// 5. NUMBER (uint32) -> 4 bytes en Big-Endian (Network Byte Order)
	if err := binary.Read(buf, binary.BigEndian, &bet.Number); err != nil {
		return nil, 0, err
	}

	return bet, len(data) - buf.Len(), nil
}

func UnmarshalBets(data []byte) ([]*Bet, error) {
	if len(data) < batchCountSize {
		return nil, fmt.Errorf("payload de ganadores incompleto")
	}

	count := binary.BigEndian.Uint16(data[:batchCountSize])
	offset := batchCountSize
	winners := make([]*Bet, 0, int(count))
	for i := uint16(0); i < count; i++ {
		bet, bytesRead, err := UnmarshalBet(data[offset:])
		if err != nil {
			return nil, err
		}
		winners = append(winners, bet)
		offset += bytesRead
	}
	if offset != len(data) {
		return nil, fmt.Errorf("payload de ganadores contiene bytes adicionales")
	}
	return winners, nil
}

func NewBetFromInputLine(line string) (*Bet, error) {
	bet := &Bet{}
	if err := ParseBetFromBytes([]byte(line), bet); err != nil {
		return nil, err
	}
	return bet, nil
}

func ParseBetFromBytes(line []byte, bet *Bet) error {
	fields := bytes.Split(line, []byte{','})
	if len(fields) != 5 {
		return fmt.Errorf("línea de entrada inválida: faltan o sobran campos")
	}

	dni, err := strconv.ParseUint(string(fields[2]), 10, 32)
	if err != nil {
		return fmt.Errorf("DNI inválido")
	}

	number, err := strconv.ParseUint(string(fields[4]), 10, 32)
	if err != nil {
		return fmt.Errorf("Número inválido")
	}

	bet.Name = string(fields[0])
	bet.LastName = string(fields[1])
	bet.Dni = uint32(dni)
	bet.Date = string(fields[3])
	bet.Number = uint32(number)

	return nil
}