package common

import (
	"bufio"
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
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
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
		<-sigc;
		if client.conn != nil {
			client.conn.Close();
		}
		os.Exit(0);
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
	nacimiento string //
	numero     uint64
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

	_, err := c.conn.Write(buf)
	return err
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()

		// TODO: Modify the send to avoid short-write
		persona := Person {
			nombre:     "Santiago Lionel",
			apellido:   "Lorca",
			dni:        "30904465",
			nacimiento: "1999-03-17",
			numero:     7574,
		}

		err := c.sendPerson(persona)
		if err != nil {
			fmt.Println("error sending person", err)
			return
		}

		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		c.conn.Close()

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)

	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
