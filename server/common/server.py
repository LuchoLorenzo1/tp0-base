import socket
import logging
import signal
from .utils import store_bets, Bet

class Person:
    def __init__(self, nombre, apellido, dni, nacimiento, numero):
        self.nombre = nombre
        self.apellido = apellido
        self.dni = dni
        self.nacimiento = nacimiento
        self.numero = numero

    def from_socket(sock: socket.socket):
        nombre_len = int.from_bytes(sock.recv(1), byteorder='big')
        nombre = sock.recv(nombre_len).decode('utf-8')

        apellido_len = int.from_bytes(sock.recv(1), byteorder='big')
        apellido = sock.recv(apellido_len).decode('utf-8')

        dni = sock.recv(8).decode('utf-8')
        nacimiento = sock.recv(10).decode('utf-8')
        numero = int.from_bytes(sock.recv(8), byteorder='big')

        return Person(nombre, apellido, dni, nacimiento, numero)

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.kill = False

        signal.signal(signal.SIGTERM, self.shutdown)

    def shutdown(self, *_, **__):
        # logging.info(f'action: shutdown | result: success')
        self.shutdown = True
        # self._server_socket.close()

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        while True:
            if self.kill:
                break
            client_sock = self.__accept_new_connection()
            if self.kill:
                break
            self.__handle_client_connection(client_sock)

        self._server_socket.close()

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            person = Person.from_socket(client_sock)

            logging.info(f'action: apuesta_almacenada | result: success | dni: {person.dni} | numero: {person.numero}')

            store_bets([Bet(0, person.nombre, person.apellido, person.dni, person.nacimiento, person.numero)])

            # TODO: Modify the send to avoid short-writes
            client_sock.send("{}\n".format("OK").encode('utf-8'))
        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
