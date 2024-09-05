from .utils import store_bets, Bet, recv_all, send_all, load_bets, has_won
import socket
import logging
import os
import threading


class LotteryProtocol:

    BET_CHUNK=1
    END_BETS=2
    GET_WINNERS=3

    READY_KEY="ready"
    AGENCIES_SET_KEY="agencias_listas"

    SUCESS=b"OK"
    ERROR=b"NO"

    def parse_bet(sock: socket.socket, agencia: int) -> Bet:
        nombre_len = int.from_bytes(recv_all(sock, 1), byteorder='big')
        nombre = recv_all(sock, nombre_len).decode('utf-8')

        apellido_len = int.from_bytes(recv_all(sock, 1), byteorder='big')
        apellido = recv_all(sock, apellido_len).decode('utf-8')

        dni = recv_all(sock, 8).decode('utf-8')
        nacimiento = recv_all(sock, 10).decode('utf-8')
        numero = int.from_bytes(recv_all(sock, 8), byteorder='big')

        return Bet(str(agencia), nombre, apellido, dni, nacimiento, str(numero))

    def get_bets(sock: socket.socket, agencia: int, bets_lock: threading.Lock):
        chunk_len = int.from_bytes(sock.recv(4), byteorder='big')
        bets = []
        for _ in range(chunk_len):
            bets.append(LotteryProtocol.parse_bet(sock, agencia))

        with bets_lock:
            store_bets(bets)

        logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")
        send_all(sock, LotteryProtocol.SUCESS)

    def send_winner(sock: socket.socket, agencia: int, state: dict, state_lock: threading.Lock):

        ready = False
        with state_lock:
            ready = state.get(LotteryProtocol.READY_KEY, False)
            if ready:
                agencias_listas: set = state.get(LotteryProtocol.AGENCIES_SET_KEY, set())
                agencias_listas.remove(agencia)
                state[LotteryProtocol.AGENCIES_SET_KEY] = agencias_listas
                if len(agencias_listas) == 0:
                    state[LotteryProtocol.READY_KEY] = False

        if not ready:
            send_all(sock, LotteryProtocol.ERROR)
            return
        else:
            send_all(sock, LotteryProtocol.SUCESS)

        winners_from_agency: list[Bet] = []
        for b in list(load_bets()):
            if has_won(b) and b.agency == agencia:
                winners_from_agency.append(b)
        send_all(sock, len(winners_from_agency).to_bytes(4, byteorder='big'))
        for winner in winners_from_agency:
            send_all(sock, winner.document.encode('utf-8'))

    def end_bets(sock: socket.socket, agencia: int, state: dict, state_lock: threading.Lock):
        logging.info(f"action: notificacion_recibida_fin_apuestas | result: success")

        with state_lock:
            agencias_listas = state.get(LotteryProtocol.AGENCIES_SET_KEY, set())
            agencias_listas.add(agencia)
            state[LotteryProtocol.AGENCIES_SET_KEY] = agencias_listas
            if len(agencias_listas) == int(os.getenv("AGENCIAS", 5)):
                state[LotteryProtocol.READY_KEY] = True

        send_all(sock, LotteryProtocol.SUCESS)

    def start(sock: socket.socket, state: dict, state_lock: threading.Lock, bets_lock: threading.Lock):
        try:
            mode = int.from_bytes(sock.recv(1), byteorder='big')
            agencia = int.from_bytes(sock.recv(4), byteorder='big')

            if mode == LotteryProtocol.BET_CHUNK:
                LotteryProtocol.get_bets(sock, agencia, bets_lock)
            elif mode == LotteryProtocol.END_BETS:
                LotteryProtocol.end_bets(sock, agencia, state, state_lock)
            elif mode == LotteryProtocol.GET_WINNERS:
                LotteryProtocol.send_winner(sock, agencia, state, state_lock)
        except Exception as e:
            logging.error(f"action: parseando_protocolo | result: fail | error: {e}")
            return None
