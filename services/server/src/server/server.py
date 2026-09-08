import socket
import threading
import logger
import safe_socket
import protocol
from lottery import Lottery

class Server:
    def __init__(self, server_host: str, server_port: int, agency_quorum_min: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.agency_quorum_min = agency_quorum_min
        self.lottery = Lottery("/tmp/bets.csv")
        self.file_lock = threading.Lock()
        self.agencias_listas = 0
        self.quorum_cond = threading.Condition()
        self.sorteo_listo = threading.Event()
        self.ganadores_memoria = {}

    def _run_lottery(self):
        with self.quorum_cond:
            self.quorum_cond.wait_for(
                lambda: self.agencias_listas >= self.agency_quorum_min
            )

        winners_by_agency = {}
        for bet in self.lottery.load_bets():
            if self.lottery.has_won(bet):
                winners_by_agency.setdefault(bet.agency_id, []).append(bet)

        self.ganadores_memoria = winners_by_agency
        self.sorteo_listo.set()

    def _handle_client(self, client_socket):
        action = "handle-client"
        message_amount = 0
        with client_socket:
            try:
                logger.info(action, logger.LogResult.in_progress)
                while True:
                    opcode, agency, client_message = safe_socket.recv_frame(client_socket)
                    message_amount += 1

                    if opcode == protocol.Opcode.SEND_BETS:
                        logger.info(action, logger.LogResult.success, "messages-amount", message_amount)
                        bets = protocol.decode_bets(client_message, agency)

                        with self.file_lock:
                            self.lottery.store_bets(bets)
                        safe_socket.send_frame(client_socket, protocol.Opcode.ACK, agency, b"")
                    if opcode == protocol.Opcode.END_BETS:
                        logger.info(action, logger.LogResult.success, "messages-amount", message_amount)
                        with self.quorum_cond:
                            self.agencias_listas += 1
                            self.quorum_cond.notify()

                        self.sorteo_listo.wait()
                        winners = self.ganadores_memoria.get(agency, [])
                        safe_socket.send_frame(
                            client_socket,
                            protocol.Opcode.WINNERS_LIST,
                            agency,
                            protocol.marshal_bets(winners),
                        )
                        return

            except Exception as e:
                logger.error(
                    action, logger.LogResult.fail, "messages-amount", message_amount
                )
                raise e

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            lottery_thread = threading.Thread(target=self._run_lottery)
            lottery_thread.start()
            client_threads = []
            try:
                while True:
                    try:
                        logger.info(action, logger.LogResult.in_progress)
                        client_socket, _ = server_socket.accept()
                    except Exception as e:
                        logger.error(action, logger.LogResult.fail)
                        raise e
                    logger.info(action, logger.LogResult.success)

                    client_thread = threading.Thread(
                        target=self._handle_client,
                        args=(client_socket,),
                    )
                    client_threads.append(client_thread)
                    client_thread.start()
            finally:
                for client_thread in client_threads:
                    client_thread.join()
                lottery_thread.join()
