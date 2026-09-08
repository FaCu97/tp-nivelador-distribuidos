package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/domain"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 15
const CONNECTION_ATTEMPS_DELAY_MS = 400

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
	BatchSize  uint16
}

type Client struct {
	conn      net.Conn
	config    ClientConfig
	shutdown  chan struct{}
	closeOnce sync.Once
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	client := &Client{
		conn:     conn,
		config:   config,
		shutdown: make(chan struct{}),
	}
	return client, nil
}

func (client *Client) Close() {
	client.closeOnce.Do(func() {
		close(client.shutdown)
		client.conn.Close()
	})
}

func (client *Client) IsShutdown() bool {
	select {
	case <-client.shutdown:
		return true
	default:
		return false
	}
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) Run() error {
	const mainAction = "test-echo-server"
	defer client.Close()

	input_file, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}
	defer input_file.Close()

	output_file, err := os.OpenFile(client.config.OutputFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("open-output-file", logger.Fail, "err", err)
		return err
	}
	defer output_file.Close()

	agency, err := strconv.ParseUint(client.config.AgencyId, 10, 8)
	if err != nil {
		return err
	}
	bet_count := uint16(0)
	scanner := bufio.NewScanner(input_file)
	bets := make([]*domain.Bet, 0)
	for scanner.Scan() {
		if client.IsShutdown() {
			return nil
		}
		clientMessage := scanner.Text()
		bet, err := domain.NewBetFromInputLine(clientMessage)
		if err != nil {
			logger.Error("parse-input-line", logger.Fail, "line", clientMessage, "err", err)
			return err
		}
		bet_count++
		bets = append(bets, bet)

		if bet_count >= client.config.BatchSize {
			payload_bets, err := domain.MarshalBets(bets)
			if err != nil {
				logger.Error("marshal-bets", logger.Fail, "err", err)
				return err
			}

			messageArgs := []any{"agency-id", client.config.AgencyId, "bets-count", bet_count}

			if err := safe_socket.SendFrame(client.conn, safe_socket.OpSendBets, byte(agency), payload_bets); err != nil {
				logger.Error("send-bets", logger.Fail, messageArgs...)
				return err
			}
			logger.Info(mainAction, logger.InProgress, messageArgs...)

			ackOpcode, _, _, err := safe_socket.RecvFrame(client.conn)
			if err != nil {
				if client.IsShutdown() {
					return nil
				}
				return err
			}

			if ackOpcode != safe_socket.OpAck {
				return fmt.Errorf("respuesta inesperada: opcode %d", ackOpcode)
			}
			bets = make([]*domain.Bet, 0)
			bet_count = 0
		}
	}
	if bet_count > 0 {
		payload_bets, err := domain.MarshalBets(bets)
		if err != nil {
			logger.Error("marshal-bets", logger.Fail, "err", err)
			return err
		}

		messageArgs := []any{"agency-id", client.config.AgencyId, "bets-count", bet_count}

		if err := safe_socket.SendFrame(client.conn, safe_socket.OpSendBets, byte(agency), payload_bets); err != nil {
			logger.Error("send-bets", logger.Fail, messageArgs...)
			return err
		}
		logger.Info(mainAction, logger.InProgress, messageArgs...)

		ackOpcode, _, _, err := safe_socket.RecvFrame(client.conn)
		if err != nil {
			if client.IsShutdown() {
				return nil
			}
			return err
		}

		if ackOpcode != safe_socket.OpAck {
			return fmt.Errorf("respuesta inesperada: opcode %d", ackOpcode)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scan-file", logger.Fail, "err", err)
		return err
	}

	if err := safe_socket.SendFrame(client.conn, safe_socket.OpEndBets, byte(agency), nil); err != nil {
		return err
	}

	opcode, _, responseBuffer, err := safe_socket.RecvFrame(client.conn)
	if err != nil {
		if client.IsShutdown() {
			return nil
		}
		logger.Error("recv-response", logger.Fail, "agency-id", client.config.AgencyId)
		return err
	}
	if opcode == safe_socket.OpWinnersList {
		winners, err := domain.UnmarshalBets(responseBuffer)
		if err != nil {
			return err
		}
		logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId, "winners-count", len(winners))
		for _, winner := range winners {
			line := fmt.Sprintf("%s,%s,%d,%s,%d\n", winner.Name, winner.LastName, winner.Dni, winner.Date, winner.Number)
			if _, err := output_file.WriteString(line); err != nil {
				logger.Error("write-output-file", logger.Fail, "err", err)
				return err
			}
		}
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)

	return nil
}
