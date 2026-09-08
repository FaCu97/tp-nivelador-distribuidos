package domain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

type Bet struct {
	Name   string
	LastName string
	Dni	uint32
	Date   string
    Number uint16
}

func (bet *Bet) MarshalBet() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	nameBytes := []byte(bet.Name)
	lastNameBytes := []byte(bet.LastName)

	if len(nameBytes) > 255 || len(lastNameBytes) > 255 {
		return nil, fmt.Errorf("nombre o apellido demasiado largos (máximo 255 bytes)")
	}

	// 1. NAME_SIZE (uint8) + NAME
	if err := buf.WriteByte(byte(len(nameBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(nameBytes); err != nil {
		return nil, err
	}

	// 2. LAST_NAME_SIZE (uint8) + LAST_NAME
	if err := buf.WriteByte(byte(len(lastNameBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(lastNameBytes); err != nil {
		return nil, err
	}

	// 3. DNI (uint32) -> 4 bytes en Big-Endian
	if err := binary.Write(buf, binary.BigEndian, bet.Dni); err != nil {
		return nil, err
	}

	// 4. DATE (string 10 bytes fijos)
	dateBytes := []byte(bet.Date)
	if len(dateBytes) != 10 {
		return nil, fmt.Errorf("la fecha debe medir exactamente 10 bytes (YYYY-MM-DD), midió %d", len(dateBytes))
	}
	buf.Write(dateBytes)

	// 5. NUMBER (uint16) -> 2 bytes en Big-Endian (Network Byte Order)
	if err := binary.Write(buf, binary.BigEndian, bet.Number); err != nil {
		return nil, err
	}

	// Retornamos el array de bytes final
	return buf.Bytes(), nil
}

func UnmarshalBet(data []byte) (*Bet, error) {
	buf := bytes.NewReader(data)
	bet := &Bet{}

	// 1. NAME_SIZE (uint8) + NAME
	var nameSize uint8
	if err := binary.Read(buf, binary.BigEndian, &nameSize); err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameSize)
	if _, err := buf.Read(nameBytes); err != nil {
		return nil, err
	}
	bet.Name = string(nameBytes)

	// 2. LAST_NAME_SIZE (uint8) + LAST_NAME
	var lastNameSize uint8
	if err := binary.Read(buf, binary.BigEndian, &lastNameSize); err != nil {
		return nil, err
	}
	lastNameBytes := make([]byte, lastNameSize)
	if _, err := buf.Read(lastNameBytes); err != nil {
		return nil, err
	}
	bet.LastName = string(lastNameBytes)

	// 3. DNI (uint32) -> 4 bytes en Big-Endian
	if err := binary.Read(buf, binary.BigEndian, &bet.Dni); err != nil {
		return nil, err
	}

	// 4. DATE (string 10 bytes fijos)
	dateBytes := make([]byte, 10)
	if _, err := buf.Read(dateBytes); err != nil {
		return nil, err
	}
	bet.Date = string(dateBytes)

	// 5. NUMBER (uint16) -> 2 bytes en Big-Endian (Network Byte Order)
	if err := binary.Read(buf, binary.BigEndian, &bet.Number); err != nil {
		return nil, err
	}

	return bet, nil
}

func UnmarshalBets(data []byte) ([]*Bet, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("payload de ganadores incompleto")
	}

	count := binary.BigEndian.Uint32(data[:4])
	offset := 4
	winners := make([]*Bet, 0, int(count))
	for i := uint32(0); i < count; i++ {
		if len(data)-offset < 4 {
			return nil, fmt.Errorf("payload de ganador incompleto")
		}
		betSize := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if len(data)-offset < betSize {
			return nil, fmt.Errorf("tamaño de ganador inválido")
		}
		bet, err := UnmarshalBet(data[offset : offset+betSize])
		if err != nil {
			return nil, err
		}
		winners = append(winners, bet)
		offset += betSize
	}
	if offset != len(data) {
		return nil, fmt.Errorf("payload de ganadores contiene bytes adicionales")
	}
	return winners, nil
}

func NewBetFromInputLine(line string) (*Bet, error) {
	parts := strings.Split(line, ",")
	if len(parts) != 5 {
		return nil, fmt.Errorf("línea de entrada inválida: %s", line)
	}

	dni, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("DNI inválido: %s", parts[2])
	}

	number, err := strconv.ParseUint(parts[4], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("Número inválido: %s", parts[4])
	}

	bet := &Bet{
		Name:     parts[0],
		LastName: parts[1],
		Dni:      uint32(dni),
		Date:     parts[3],
		Number:   uint16(number),
	}

	return bet, nil
}