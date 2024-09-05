import os
import socket
import logging
import signal
from .lottery import LotteryProtocol
import threading

class Server:
    def __init__(self, port, listen_backlog):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.kill = False

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

        threads = []
        max_threads = int(os.getenv('AGENCIAS', 3))

        state = dict()
        state_lock = threading.Lock()
        bets_lock = threading.Lock()

        while True:
            if self.kill:
                break
            
            client_sock = self.__accept_new_connection()

            thread = threading.Thread(target=self.__handle_client_connection, args=(client_sock,state,state_lock,bets_lock))
            thread.start()
            threads.append(thread)

            threads = [t for t in threads if t.is_alive()]

            while len(threads) >= max_threads:
                for t in threads:
                    t.join(timeout=0.1)
                threads = [t for t in threads if t.is_alive()]

        logging.info(f'action: shutdown | result: success')
        self._server_socket.close()

    def __handle_client_connection(self, client_sock, state, state_lock, bets_lock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            LotteryProtocol.start(client_sock, state, state_lock, bets_lock)
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
