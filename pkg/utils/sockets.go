package utils

import (
	"encoding/json"
	"net"

	"github.com/gobwas/ws/wsutil"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

func ReadClientMessage[T dto.Dto](conn *net.Conn) (*dto.ClientMessageDto[T], error) {
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

func WriteClientMessage[T dto.Dto](conn *net.Conn, message T) error {
	connInstance := *conn
	clientMessageDto := dto.ClientMessageDto[T]{
		Data: message,
	}
	b, err := json.Marshal(clientMessageDto)
	if err != nil {
		return err
	}
	if err := wsutil.WriteClientBinary(connInstance, b); err != nil {
		return err
	}
	return nil
}

func ReadServerMessage[T dto.Dto](conn *net.Conn) (*dto.ServerMessageDto[T], error) {
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

func WriteServerError[T dto.Dto](conn *net.Conn, message string) error {
	return writeToServer(conn, dto.ServerMessageDto[T]{
		ErrorMessage: message,
		Error:        true,
	})
}

func WriteServerMessage[T dto.Dto](conn *net.Conn, data T) error {
	return writeToServer(conn, dto.ServerMessageDto[T]{
		Data:  data,
		Error: false,
	})
}

func writeToServer[T dto.Dto](conn *net.Conn, data dto.ServerMessageDto[T]) error {
	connInstance := *conn
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := wsutil.WriteServerBinary(connInstance, b); err != nil {
		return err
	}
	return nil
}
