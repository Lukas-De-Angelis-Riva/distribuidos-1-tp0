import socket
import logging
import signal

import time

from common.protocol import recv_bet, confirm_bet
from common.utils import store_bets

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

        self._keep_running = True
        signal.signal(signal.SIGTERM, self.__stop)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        # TODO: Modify this program to handle signal to graceful shutdown
        # the server
        while self._keep_running:
            client_sock = self.__accept_new_connection()
            if client_sock:
                self.__handle_client_connection(client_sock)

        logging.info('action: stop_server | result: success')

    def __stop(self, *args):
        logging.info('action: stop_server | result: in_progress')
        self._keep_running = False
        self._server_socket.shutdown(socket.SHUT_RDWR)
        logging.info('action: shutdown_socket | result: success')
        self._server_socket.close()
        logging.info('action: closing_socket | result: success')

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """

        try:
            bet = recv_bet(client_sock)
            store_bets([bet])
            logging.info(f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}")

            confirm_bet(client_sock, str(bet.number))

        except Exception as e:
            logging.error(f"action: recibir_apuesta | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        try:
            logging.info('action: accept_connections | result: in_progress')
            c, addr = self._server_socket.accept()
            logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        except OSError as e:
            logging.info('action: accept_connections | result: fail')
            return False

        return c
