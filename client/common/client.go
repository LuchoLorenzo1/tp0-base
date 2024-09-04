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

const BET_CHUNK=1
const END_BETS=2
const GET_WINNERS=3


func (c *Client) StartClientLoop() {
	c._StartClientLoop()
	if c.conn != nil {
		c.conn.Close()
	}
}

func recv_all(sock net.Conn, sz int) []byte {
	buffer := make([]byte, sz)
	totalRead := 0
	for totalRead < len(buffer) {
		size, err := sock.Read(buffer)
		if err != nil {
			fmt.Errorf("failed to read data: %w. Trying again.", err)
		}
		totalRead += size
	}
	return buffer
}

func send_agency_number(sock net.Conn, agencia uint32) error {
	numberOfBatchesBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(numberOfBatchesBytes, agencia)
	return send_all(sock, numberOfBatchesBytes)
}

func (c *Client) _StartClientLoop() {
	agencia := uint32(c.config.AgencyNumber)
	records := readCsvFile(c.config.AgencyFile)
	i := 0

	if MAX_SIZE_PERSON*c.config.BatchMaxAmount > 8000 {
		c.config.BatchMaxAmount = 8000 / MAX_SIZE_PERSON
	}

	for i < len(records) {
		c.createClientSocket()
		send_all(c.conn, []byte{BET_CHUNK})

		err := send_agency_number(c.conn, agencia)
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

		buffer = recv_all(c.conn, 2)
		c.conn.Close()

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

	c.createClientSocket()
	err := send_all(c.conn, []byte{END_BETS})
	if err != nil {
		fmt.Errorf("error mode message", err)
		return
	}
	err = send_agency_number(c.conn, agencia)
	if err != nil {
		fmt.Println("error sending agency number", err)
		return
	}

	buffer := recv_all(c.conn, 2)
	c.conn.Close()
	response := string(buffer)
	if response != "OK" {
		fmt.Errorf("error sending person")
		return
	}

	a := true
	for a {
		c.createClientSocket()
		if c.conn == nil {
			time.Sleep(c.config.LoopPeriod)
			continue
		}

		send_all(c.conn, []byte{GET_WINNERS})
		if err != nil {
			fmt.Errorf("error mode message", err)
			return
		}
		err = send_agency_number(c.conn, agencia)
		if err != nil {
			fmt.Println("error sending agency number", err)
			return
		}

		buffer = recv_all(c.conn, 2)
		response = string(buffer)

		if response != "OK" {
			c.conn.Close()
			time.Sleep(c.config.LoopPeriod)
			continue
		} else {
			a = false
		}
	} 
	
	numOfDocuments := binary.BigEndian.Uint32(recv_all(c.conn, 4))
	for i := 0; i < int(numOfDocuments); i++ {
		recv_all(c.conn, 8)
	}

	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", numOfDocuments)
	c.conn.Close()
}
