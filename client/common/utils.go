package common

import (
	"fmt"
	"encoding/csv"
	"os"
	"net"
)

// Send all data in the buffer to the socket, preventing short writes
func send_all(sock net.Conn, buf []byte) error {
	totalWritten := 0
	for totalWritten < len(buf) {
		size, err := sock.Write(buf[totalWritten:])
		if err != nil {
			return fmt.Errorf("failed to write data: %w. Trying again.", err)
		}
		totalWritten += size
	}
	return nil
}

// Read a CSV file and return its content as a 2D array of strings
func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}
	return records
}
