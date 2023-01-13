package dto

import "github.com/google/uuid"

type Dto interface {
	DataDto | UserDto | UserMachineDto | ChallengeResponseDto | ChallengeSuccessEncryptedKeyDto | MessageDto | EncryptedMasterKeyDto | MasterKeyDto | PublicKeyDto
}

type DataDto struct {
	ID        uuid.UUID      `json:"id"`
	Username  string         `json:"username"`
	Keys      []KeyDto       `json:"keys"`
	MasterKey []byte         `json:"master_key"`
	SshConfig []SshConfigDto `json:"ssh_config"`
	Machines  []MachineDto   `json:"machines"`
}

type KeyDto struct {
	ID       uuid.UUID `json:"id"`
	UserID   uuid.UUID `json:"user_id"`
	Filename string    `json:"filename"`
	Data     []byte    `json:"data"`
}

type SshConfigDto struct {
	Host         string            `json:"host"`
	Values       map[string]string `json:"values"`
	IdentityFile string            `json:"identity_file"`
}

type MachineDto struct {
	Name string `json:"machine_name"`
}

type UserDto struct {
	Username string       `json:"username"`
	Machines []MachineDto `json:"machines"`
}

type UserMachineDto struct {
	Username    string `json:"username"`
	MachineName string `json:"machine_name"`
}

type ChallengeResponseDto struct {
	Challenge string `json:"challenge"`
}

type ChallengeSuccessEncryptedKeyDto struct {
	EncryptedMasterKey []byte `json:"encrypted_master_key"`
	PublicKey          []byte `json:"public_key"`
}

type MessageDto struct {
	Message string `json:"message"`
}

type EncryptedMasterKeyDto struct {
	EncryptedMasterKey []byte `json:"encrypted_master_key"`
}

type MasterKeyDto struct {
	MasterKey []byte `json:"master_key"`
}

type PublicKeyDto struct {
	PublicKey []byte `json:"public_key"`
}

type ServerMessageDto[T Dto] struct {
	Data         T      `json:"data"`
	Error        bool   `json:"error"`
	ErrorMessage string `json:"error_message"`
}

type ClientMessageDto[T Dto] struct {
	Message string `json:"message"`
	Data    T      `json:"data"`
}
