package common

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"encoding/csv"

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
	reader *csv.Reader
	file   *os.File	
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
		if client.file != nil {
			client.file.Close()
		}
		os.Exit(0)
	}()

	return client
}

func (c *Client) createClientSocket() error {
	var conn net.Conn = nil
	var err error = fmt.Errorf("")
	retries := 5
	for conn == nil && retries > 0 {
		conn, err = net.Dial("tcp", c.config.ServerAddress)
		if err != nil {
			log.Criticalf(
				"action: connect | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			retries--
			time.Sleep(2 * c.config.LoopPeriod)
		} else {
			break
		}
	}

	if retries <= 0 {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		c.conn = nil
		return err
	}
	

	c.conn = conn

	return nil
}

const BET_CHUNK=1
const END_BETS=2
const GET_WINNERS=3



// Read a CSV file and return its content as a 2D array of strings
func (c *Client) openCsvReader(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}

	c.file = f
	c.reader = csv.NewReader(f)
	return nil
}

func (c *Client) StartClientLoop() {
	c._StartClientLoop()

	if c.conn != nil {
		c.conn.Close()
	}
	if c.file != nil {
		c.file.Close()
	}
}

func (c *Client) _StartClientLoop() {
	agencia := uint32(c.config.AgencyNumber)
	err := c.openCsvReader(c.config.AgencyFile)
	if err != nil || c.reader == nil {
		log.Errorf("error opening csv file", err)
		return
	}

	i := 0

	if MAX_SIZE_PERSON*c.config.BatchMaxAmount > 8000 {
		c.config.BatchMaxAmount = 8000 / MAX_SIZE_PERSON
	}

	coninue := true
	for coninue {
		c.createClientSocket()
		if c.conn == nil {
			log.Errorf("error creating client socket")
			return
		}

		err := send_all(c.conn, []byte{BET_CHUNK})
		if err != nil {
			log.Errorf("action: send_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}

		err = send_agency_number(c.conn, agencia)
		if err != nil {
			log.Errorf("error sending agency number", err)
			return
		}

		chunkPersonas := make([]*Person, 0, c.config.BatchMaxAmount)
		for len(chunkPersonas) < c.config.BatchMaxAmount {
			record, err := c.reader.Read()
			if err != nil {
				if err.Error() == "EOF" {
					coninue = false
					break
				} 
				log.Errorf("error reading csv file", err)
				return
			}

			persona, err := person_from_string_array(record)
			if err != nil {
				fmt.Println("error creating person", err)
				return
			}
			chunkPersonas = append(chunkPersonas, persona)
			i++
		}

		if len(chunkPersonas) == 0 {
			break
		}

		buffer := make([]byte, 4)
		binary.BigEndian.PutUint32(buffer, uint32(len(chunkPersonas)))

		for _, persona := range chunkPersonas {
			buffer, err = serializePerson(buffer, *persona) 
			if err != nil {
				log.Errorf("error sending person", err)
				return
			}
		}

		err = send_all(c.conn, buffer)
		if err != nil {
			log.Errorf("action: send_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}

		buffer, err = recv_all(c.conn, 2)
		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}
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
	if c.conn == nil {
		fmt.Errorf("error creating client socket")
		return
	}
	err = send_all(c.conn, []byte{END_BETS})
	if err != nil {
		log.Errorf("action: send_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	err = send_agency_number(c.conn, agencia)
	if err != nil {
		fmt.Println("error sending agency number", err)
		return
	}

	buffer, err := recv_all(c.conn, 2)
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

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
			fmt.Errorf("error creating client socket")
			return
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

		buffer, err = recv_all(c.conn, 2)
		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}


		response = string(buffer)

		if response != "OK" {
			c.conn.Close()
			time.Sleep(c.config.LoopPeriod)
			continue
		} else {
			a = false
		}
	} 

	buffer, err = recv_all(c.conn, 4)
	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	
	numOfDocuments := binary.BigEndian.Uint32(buffer)
	for i := 0; i < int(numOfDocuments); i++ {
		_, err = recv_all(c.conn, 8)
		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}
	}

	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", numOfDocuments)
	c.conn.Close()
}
