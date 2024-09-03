package common

import (
	"encoding/binary"
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
	buf := make([]byte, 0, 1)

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

	return send_all(c.conn, buf)
}

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

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	retries := 0
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Create the connection the server in every loop iteration. Send an
		err := c.createClientSocket()
		if c.conn == nil || err != nil {
			if retries < 5 {
				retries++
				time.Sleep(c.config.LoopPeriod)
				continue
			}
			os.Exit(1)
			return
		}

		persona := Person {
			nombre:     os.Getenv("NOMBRE") ,
			apellido:   os.Getenv("APELLIDO"),
			dni:        os.Getenv("DNI"),
			nacimiento: os.Getenv("NACIMIENTO"),
		}
		i := os.Getenv("NUMERO")
		v, err := strconv.ParseUint(i, 10, 64)
		if err != nil {
			persona.numero = 7777
		} else {
			persona.numero = v
		}

		err = c.sendPerson(persona)
		if err != nil {
			return
		}
		log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s", persona.dni, persona.numero)

		buffer := make([]byte, 2)
		totalRead := 0
		for totalRead < len(buffer) {
			size, err := c.conn.Read(buffer)
			log.Infof("size READ %v", size)
			if err != nil {
				fmt.Errorf("failed to read data: %w. Trying again.", err)
			}
			totalRead += size
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
			string(buffer),
		)

		time.Sleep(c.config.LoopPeriod)
	}

	if c.conn != nil {
		c.conn.Close()
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
