import socket
import signal
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
        self.shutdown_event = threading.Event()
        self.client_sockets = set()
        self.client_sockets_lock = threading.Lock()
        self.ganadores_memoria = {}

    def _handle_sigterm(self, signum, frame):
        self.shutdown_event.set()
        with self.quorum_cond:
            self.quorum_cond.notify_all()
        self.sorteo_listo.set()

    def _close_client_sockets(self):
        with self.client_sockets_lock:
            client_sockets = list(self.client_sockets)

        for client_socket in client_sockets:
            try:
                client_socket.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            client_socket.close()

    def _run_lottery(self):
        with self.quorum_cond:
            self.quorum_cond.wait_for(
                lambda: self.shutdown_event.is_set()
                or self.agencias_listas >= self.agency_quorum_min
            )

        if self.shutdown_event.is_set():
            return

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
                while not self.shutdown_event.is_set():
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

                        while not self.shutdown_event.is_set():
                            if self.sorteo_listo.wait(timeout=0.5):
                                break

                        if self.shutdown_event.is_set():
                            return

                        winners = self.ganadores_memoria.get(agency, [])
                        safe_socket.send_frame(
                            client_socket,
                            protocol.Opcode.WINNERS_LIST,
                            agency,
                            protocol.marshal_bets(winners),
                        )
                        return

            except Exception as e:
                if self.shutdown_event.is_set():
                    return
                logger.error(
                    action, logger.LogResult.fail, "messages-amount", message_amount
                )
                raise e
            finally:
                with self.client_sockets_lock:
                    self.client_sockets.discard(client_socket)

    def run(self):
        action = "accept-connection"
        signal.signal(signal.SIGTERM, self._handle_sigterm)
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.settimeout(0.5)
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            lottery_thread = threading.Thread(target=self._run_lottery)
            lottery_thread.start()
            client_threads = []
            try:
                while not self.shutdown_event.is_set():
                    try:
                        logger.info(action, logger.LogResult.in_progress)
                        client_socket, _ = server_socket.accept()
                    except socket.timeout:
                        continue
                    except Exception as e:
                        if self.shutdown_event.is_set():
                            break
                        logger.error(action, logger.LogResult.fail)
                        raise e
                    logger.info(action, logger.LogResult.success)

                    with self.client_sockets_lock:
                        self.client_sockets.add(client_socket)

                    client_thread = threading.Thread(
                        target=self._handle_client,
                        args=(client_socket,),
                    )
                    client_threads.append(client_thread)
                    client_thread.start()
            finally:
                self.shutdown_event.set()
                with self.quorum_cond:
                    self.quorum_cond.notify_all()
                self.sorteo_listo.set()
                self._close_client_sockets()
                for client_thread in client_threads:
                    client_thread.join()
                lottery_thread.join()
