package utils

import (
	"encoding/json"
	"net"

	"github.com/gobwas/ws/wsutil"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

func ReadServerMessage[T any](conn *net.Conn) (*dto.ServerMessageDto[T], error) {
	connInstance := *conn
	data, err := wsutil.ReadServerBinary(connInstance)
	if err != nil {
		return nil, err
	}
	var serverMessageDto dto.ServerMessageDto[T]
	if err := json.Unmarshal(data, &serverMessageDto); err != nil {
		return nil, err
	}
	return &serverMessageDto, nil
}

func ReadClientMessage[T any](conn *net.Conn) (*dto.ClientMessageDto[T], error) {
	connInstance := *conn
	data, err := wsutil.ReadClientBinary(connInstance)
	if err != nil {
		return nil, err
	}
	var clientMessageDto dto.ClientMessageDto[T]
	if err := json.Unmarshal(data, &clientMessageDto); err != nil {
		return nil, err
	}
	return &clientMessageDto, nil
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

func WriteServerMessage[T any](conn *net.Conn, data T, message string, isError bool) error {
	connInstance := *conn
	msg := dto.ServerMessageDto[T]{
		Data:    data,
		Message: message,
		Error:   isError,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := wsutil.WriteServerBinary(connInstance, b); err != nil {
		return err
	}
	return nil
}
