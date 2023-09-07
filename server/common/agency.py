import logging
import threading

import common
from common.protocol import recv_req, confirm_req, force_to_wait, notify_winners
from common.utils import store_bets, load_bets, has_won
from common.counter import Counter

class Agency(threading.Thread):
    def __init__(self, client_sock, bets_file_lock: threading.Lock, processed_agencies: Counter, processed_agencies_lock: threading.Lock):
        threading.Thread.__init__(self)
        self.client_sock = client_sock
        self.bets_file_lock = bets_file_lock
        self.processed_agencies = processed_agencies
        self.processed_agencies_lock = processed_agencies_lock

    def run(self):
        """
        Loop de la agencia. Lee las solicitudes del cliente, estas pueden ser:
            * Enviar apuestas o chunks de apuestas
            * Finalizar el envio de apuestas
            * Solicitar los ganadores

        Coordina la conexion con el cliente y hace uso de los mecanismos de sincronismo entre otras agencias,
        sobre el archivo de apuestas y el contador de agencias que terminaron su procesamiento. 
        """
        while True:
            try:
                req, data = recv_req(self.client_sock)
                if req == common.protocol.UPLOAD_BETS_REQ:
                    bets = data

                    self.bets_file_lock.acquire()
                    store_bets(bets)
                    self.bets_file_lock.release()

                    logging.info(f"action: request_processed | result: success | client: {self.client_sock.getpeername()[0]}")
                    confirm_req(self.client_sock)

                elif req == common.protocol.FINISH_REQ:
                    logging.info(f"action: finish_processing | result: success | client: {self.client_sock.getpeername()[0]}")

                    self.processed_agencies_lock.acquire()
                    self.processed_agencies.inc()
                    if not self.processed_agencies.less_than(5):
                        logging.info(f"action: sorteo | result: success")
                    self.processed_agencies_lock.release()
                    break

                elif req == common.protocol.POLL_WINNERS_REQ:
                    agency_number = data

                    self.processed_agencies_lock.acquire()
                    should_wait = self.processed_agencies.less_than(5)
                    self.processed_agencies_lock.release()

                    if not should_wait:
                        winners = self.__getWinners(agency_number)
                        notify_winners(self.client_sock, winners)
                        break # goodbye
                    else:
                        force_to_wait(self.client_sock)
                        break # goodbye

            except Exception as e:
                logging.error(f"action: request_processed | result: fail | error: {e}")
                break

        self.client_sock.close()

    def __getWinners(self, agency_number):
        """
        Funcion que, dado un numero de agencia, devuelve los documentos
        de todos aquellos participantes que ganaron el sorteo
        """
        self.bets_file_lock.acquire()
        winners = []
        for bet in load_bets():
            if has_won(bet) and agency_number == int(bet.agency):
                winners.append(bet.document)
        self.bets_file_lock.release()
        return winners