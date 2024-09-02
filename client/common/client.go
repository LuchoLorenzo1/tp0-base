package common

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID             string
	ServerAddress  string
	LoopAmount     int
	LoopPeriod     time.Duration
	BatchMaxAmount int
	AgencyNumber   int
	AgencyFile     string
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM)
	go func() {
		<-sigc
		if client.conn != nil {
			client.conn.Close()
		}
		os.Exit(0)
	}()

	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

type Person struct {
	nombre     string // max length 23
	apellido   string // max length 23
	dni        string // max length 8
	nacimiento string // max length 10
	numero     uint64 // 8 bytes
}

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

const MAX_NAME_LENGTH = 23
const MAX_SURNAME_LENGTH = 10
const MAX_SIZE_PERSON = 1 + MAX_NAME_LENGTH + 1 + MAX_SURNAME_LENGTH + 8 + 10 + 8

func (c *Client) sendPerson(persona Person) error {
	buf := make([]byte, 0, MAX_SIZE_PERSON)

	nameLength := len(persona.nombre)
	if nameLength > MAX_NAME_LENGTH {
		return fmt.Errorf("name too long")
	}

	buf = append(buf, byte(nameLength))
	buf = append(buf, []byte(persona.nombre)...)

	surnameLength := len(persona.apellido)
	if surnameLength > 10 {
		return fmt.Errorf("surname too long")
	}
	buf = append(buf, byte(surnameLength))
	buf = append(buf, []byte(persona.apellido)...)

	buf = append(buf, []byte(persona.dni)...)
	buf = append(buf, []byte(persona.nacimiento)...)

	number := make([]byte, 8)
	binary.BigEndian.PutUint64(number, persona.numero)
	buf = append(buf, number...)

	totalWritten := 0
	for totalWritten < len(buf) {
		size, err := c.conn.Write(buf[totalWritten:])
		if err != nil {
			return fmt.Errorf("failed to write data: %w. Trying again.", err)
		}
		totalWritten += size
	}

	return nil
}

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

func (c *Client) StartClientLoop() {
	agencia := uint32(c.config.AgencyNumber)
	records := readCsvFile(c.config.AgencyFile)
	i := 0

	if MAX_SIZE_PERSON*c.config.BatchMaxAmount > 8000 {
		c.config.BatchMaxAmount = 8000 / MAX_SIZE_PERSON
	}

	for i < len(records) {
		c.createClientSocket()

		numberOfBatchesBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(numberOfBatchesBytes, agencia)
		_, err := c.conn.Write(numberOfBatchesBytes)
		if err != nil {
			fmt.Println("error sending agency number", err)
			return
		}
		log.Info("sent agency number")

		chunkPersonas := make([]*Person, 0, c.config.BatchMaxAmount)

		for len(chunkPersonas) < c.config.BatchMaxAmount && i < len(records) {
			persona, err := person_from_string_array(records[i])
			if err != nil {
				fmt.Println("error creating person", err)
				return
			}
			chunkPersonas = append(chunkPersonas, persona)
			i++
		}

		numberOfBatchesBytes = make([]byte, 4)
		binary.BigEndian.PutUint32(numberOfBatchesBytes, uint32(len(chunkPersonas)))
		_, err = c.conn.Write(numberOfBatchesBytes)

		for _, persona := range chunkPersonas {
			err := c.sendPerson(*persona)
			if err != nil {
				fmt.Println("error sending person", err)
				return
			}
		}


		buffer := make([]byte, 2)
		totalRead := 0
		for totalRead < len(buffer) {
			size, err := c.conn.Read(buffer)
			if err != nil {
				fmt.Errorf("failed to read data: %w. Trying again.", err)
			}
			totalRead += size
		}

		response := string(buffer)
		if response != "OK" {
			fmt.Println("error sending person")
			return
		}

		log.Info("RESPUESTA", response)

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			response,
		)

		time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
	log.Infof("i: ", i)
}
