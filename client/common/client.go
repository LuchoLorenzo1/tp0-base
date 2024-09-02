package common

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
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

		buffer := make([]byte, 4)
		binary.BigEndian.PutUint32(buffer, uint32(len(chunkPersonas)))

		for _, persona := range chunkPersonas {
			buffer, err = serializePerson(buffer, *persona) 
			if err != nil {
				fmt.Errorf("error sending person", err)
				return
			}
		}

		err = send_all(c.conn, buffer)
		if err != nil {
			fmt.Errorf("error sending message", err)
			return
		}

		buffer = make([]byte, 2)
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
			fmt.Errorf("error sending person")
			return
		}

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
}
