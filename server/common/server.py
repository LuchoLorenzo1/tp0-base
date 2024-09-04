import socket
import logging
import signal
from .lottery import LotteryProtocol
import concurrent.futures

class Server:
    def __init__(self, port, listen_backlog):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.kill = False
        self.state = dict()

        signal.signal(signal.SIGTERM, self.shutdown)

    def shutdown(self, *_, **__):
        self.shutdown = True

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        with concurrent.futures.ThreadPoolExecutor(5) as executor:
            while True:
                if self.kill:
                    break
                client_sock = self.__accept_new_connection()
                executor.submit(self.__handle_client_connection, client_sock)

        logging.info(f'action: shutdown | result: success')
        self._server_socket.close()

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            LotteryProtocol.start(client_sock, self.state)
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
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
