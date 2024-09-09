package common

import (
	"fmt"
	"net"
	"encoding/binary"
)

// Send all data in the buffer to the socket, preventing short writes
func send_all(sock net.Conn, buf []byte) error {
	totalWritten := 0
	for totalWritten < len(buf) {
		size, err := sock.Write(buf[totalWritten:])
		if err != nil {
			return fmt.Errorf("failed to write data: %w.", err)
		}
		totalWritten += size
	}
	return nil
}

func recv_all(sock net.Conn, sz int) ([]byte, error) {
	buffer := make([]byte, sz)
	totalRead := 0
	for totalRead < len(buffer) {
		size, err := sock.Read(buffer)
		if err != nil {
			return nil, fmt.Errorf("failed to read data: %w. Trying again.", err)
		}
		totalRead += size
	}
	return buffer, nil
}

func send_agency_number(sock net.Conn, agencia uint32) error {
	numberOfBatchesBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(numberOfBatchesBytes, agencia)
	return send_all(sock, numberOfBatchesBytes)
}
