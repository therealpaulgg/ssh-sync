package utils

import (
	"encoding/json"
	"net"

	"github.com/gobwas/ws/wsutil"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

func ReadServerMessage(conn *net.Conn) (*dto.ServerMessageDto, error) {
	connInstance := *conn
	data, err := wsutil.ReadServerBinary(connInstance)
	if err != nil {
		return nil, err
	}
	var serverMessageDto dto.ServerMessageDto
	if err := json.Unmarshal(data, &serverMessageDto); err != nil {
		return nil, err
	}
	return &serverMessageDto, nil
}

func WriteClientMessage[T any](conn *net.Conn, message T) error {
	connInstance := *conn
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if err := wsutil.WriteClientBinary(connInstance, b); err != nil {
		return err
	}
	return nil
}
