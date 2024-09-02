package common

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

type Person struct {
	nombre     string // max length 23
	apellido   string // max length 23
	dni        string // max length 8
	nacimiento string // max length 10
	numero     uint64 // 8 bytes
}

const MAX_NAME_LENGTH = 23
const MAX_SURNAME_LENGTH = 10
const MAX_SIZE_PERSON = 1 + MAX_NAME_LENGTH + 1 + MAX_SURNAME_LENGTH + 8 + 10 + 8

func person_from_string_array(records []string) (*Person, error) {
	persona1 := &Person{
		nombre:     records[0],
		apellido:   records[1],
		dni:        records[2],
		nacimiento: records[3],
	}
	numero, err := strconv.Atoi(records[4])
	if err != nil {
		return nil, err
	}
	persona1.numero = uint64(numero)

	return persona1, nil
}

func serializePerson(buf []byte, persona Person) ([]byte, error) {
	nameLength := len(persona.nombre)
	if nameLength > MAX_NAME_LENGTH {
		return buf, fmt.Errorf("name too long")
	}

	buf = append(buf, byte(nameLength))
	buf = append(buf, []byte(persona.nombre)...)

	surnameLength := len(persona.apellido)
	if surnameLength > 10 {
		return buf, fmt.Errorf("surname too long")
	}
	buf = append(buf, byte(surnameLength))
	buf = append(buf, []byte(persona.apellido)...)

	buf = append(buf, []byte(persona.dni)...)
	buf = append(buf, []byte(persona.nacimiento)...)

	number := make([]byte, 8)
	binary.BigEndian.PutUint64(number, persona.numero)
	buf = append(buf, number...)

	return buf, nil
}
