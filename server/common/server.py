import socket
import logging
import signal

import time

import common
from common.protocol import recv_req, confirm_req, force_to_wait, notify_winners
from common.utils import store_bets, load_bets, has_won

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

        self._keep_running = True
        signal.signal(signal.SIGTERM, self.__stop)
        self.bet_amount = 0
        self.processed_agencies = 0

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

    def __getWinners(self, agency_number):
        winners = []
        for bet in load_bets():
            if has_won(bet) and agency_number == int(bet.agency):
                winners.append(bet.document)
        return winners

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """

        while True:
            try:
                req, data = recv_req(client_sock)
                if req == common.protocol.UPLOAD_BETS_REQ:
                    bets = data
                    store_bets(bets)
                    self.bet_amount += len(bets)
                    logging.info(f"action: request_processed | result: success | client: {client_sock.getpeername()[0]}")
                    confirm_req(client_sock)

                elif req == common.protocol.FINISH_REQ:
                    logging.info(f"action: finish_processing | result: success | client: {client_sock.getpeername()[0]} | bet_amount: {self.bet_amount}")
                    self.processed_agencies += 1 # REF: maybe a set, to prevent malicious clients
                    if self.processed_agencies == 5:
                        logging.info(f"action: sorteo | result: success")
                    break

                elif req == common.protocol.POLL_WINNERS_REQ:
                    agency_number = data

                    if self.processed_agencies == 5:
                        winners = self.__getWinners(agency_number)
                        notify_winners(client_sock, winners)
                        break # goodbye
                    else:
                        force_to_wait(client_sock)
                        break # goodbye

            except Exception as e:
                logging.error(f"action: request_processed | result: fail | error: {e}")
                break

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
