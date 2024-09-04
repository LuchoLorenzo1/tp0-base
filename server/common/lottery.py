from .utils import store_bets, Bet, recv_all, send_all, load_bets, has_won
import socket
import logging
import os

class LotteryProtocol:

    BET_CHUNK=1
    END_BETS=2
    GET_WINNERS=3

    def parse_bet(sock: socket.socket, agencia: int) -> Bet:
        nombre_len = int.from_bytes(recv_all(sock, 1), byteorder='big')
        nombre = recv_all(sock, nombre_len).decode('utf-8')

        apellido_len = int.from_bytes(recv_all(sock, 1), byteorder='big')
        apellido = recv_all(sock, apellido_len).decode('utf-8')

        dni = recv_all(sock, 8).decode('utf-8')
        nacimiento = recv_all(sock, 10).decode('utf-8')
        numero = int.from_bytes(recv_all(sock, 8), byteorder='big')

        return Bet(str(agencia), nombre, apellido, dni, nacimiento, str(numero))

    def get_bets(sock, agencia):
        chunk_len = int.from_bytes(sock.recv(4), byteorder='big')
        bets = []
        for _ in range(chunk_len):
            bets.append(LotteryProtocol.parse_bet(sock, agencia))
        store_bets(bets)
        logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")
        send_all(sock, b"OK")

    def send_winner(sock, agencia):
        winners_from_agency: list[Bet] = []
        for b in list(load_bets()):
            if has_won(b) and b.agency == agencia:
                winners_from_agency.append(b)
        send_all(sock, len(winners_from_agency).to_bytes(4, byteorder='big'))
        for winner in winners_from_agency:
            send_all(sock, winner.document.encode('utf-8'))

    def start(sock: socket.socket, state: dict) -> list[Bet]:
        try:
            mode = int.from_bytes(sock.recv(1), byteorder='big')
            agencia = int.from_bytes(sock.recv(4), byteorder='big')

            if mode == LotteryProtocol.BET_CHUNK:
                LotteryProtocol.get_bets(sock, agencia)
            elif mode == LotteryProtocol.END_BETS:
                logging.info(f"action: notificacion_recibida_fin_apuestas | result: success")

                agencias_listas = state.get("agencias_listas", set())
                agencias_listas.add(agencia)
                state["agencias_listas"] = agencias_listas
                if len(agencias_listas) == (int(os.getenv("AGENCIAS")) or 5):
                    state["ready"] = True

                send_all(sock, b"OK")
            elif mode == LotteryProtocol.GET_WINNERS:
                if not state.get("ready", False):
                    send_all(sock, b"NO")
                    return
                agencias_listas: set = state.get("agencias_listas", set())
                agencias_listas.remove(agencia)
                state["agencias_listas"] = agencias_listas
                if len(agencias_listas) == 0:
                    state["ready"] = False
                send_all(sock, b"OK")
                LotteryProtocol.send_winner(sock, agencia)
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | error: {e}")
            return None
