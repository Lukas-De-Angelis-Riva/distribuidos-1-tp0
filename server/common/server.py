import socket
import logging
import signal
import threading

from common.agency import Agency
from common.counter import Counter
class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

        self._keep_running = True
        signal.signal(signal.SIGTERM, self.__stop)

        self.bets_file_lock = threading.Lock()

        self.processed_agencies = Counter(0)
        self.processed_agencies_lock = threading.Lock()

    def __joinFinishedAgencies(self, agencies):
        finished_agencies = []
        for agency in agencies:
            if not agency.is_alive():
                finished_agencies.append(agency)
        for f_agency in finished_agencies:
            f_agency.join()
            logging.info('action: join finished agency | result: success')
            agencies.remove(f_agency)

    def __stopUnfinishedAgencies(self, agencies):
        for agency in agencies:
            agency.stop()
            agency.join()

    def run(self):
        agencies = []
        while self._keep_running:
            client_sock = self.__accept_new_connection()
            if client_sock:
                agency = Agency(client_sock, self.bets_file_lock,
                    self.processed_agencies, self.processed_agencies_lock)
                agency.start()
                agencies.append(agency)

            self.__joinFinishedAgencies(agencies)

        # Si quedo alguna agencia viva, se le hace stop y luego se hace join para que no queden zombies
        self.__stopUnfinishedAgencies(agencies)

        logging.info('action: stop_server | result: success')

    def __stop(self, *args):
        logging.info('action: stop_server | result: in_progress')
        self._keep_running = False
        self._server_socket.shutdown(socket.SHUT_RDWR)
        logging.info('action: shutdown_socket | result: success')
        self._server_socket.close()
        logging.info('action: closing_socket | result: success')

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
