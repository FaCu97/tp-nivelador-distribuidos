import socket
import logger
import safe_socket
import protocol
from lottery import Lottery

class Server:
    def __init__(self, server_host: str, server_port: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.lottery = Lottery("/tmp/bets.csv")

    def _handle_client(self, client_socket):
        action = "handle-client"
        message_amount = 0
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                opcode, agency, client_message = safe_socket.recv_frame(client_socket)
                message_amount += 1

                if opcode == protocol.Opcode.SEND_BET:
                    logger.info(action, logger.LogResult.success, "messages-amount", message_amount)
                    bet = protocol.decode_bet(client_message, agency)
                    self.lottery.store_bets([bet])
                if opcode == protocol.Opcode.END_BETS:
                    logger.info(action, logger.LogResult.success, "messages-amount", message_amount)
                    bets = self.lottery.load_bets()
                    winners = [
                        bet
                        for bet in self.lottery.load_bets()
                        if bet.agency_id == agency and self.lottery.has_won(bet)
                    ]
                    safe_socket.send_frame(
                        client_socket,
                        protocol.Opcode.WINNERS_LIST,
                        agency,
                        protocol.marshal_bets(winners),
                    )
                    return
               # safe_socket.send_frame(client_socket, opcode, agency, client_message)
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
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                self._handle_client(client_socket)
