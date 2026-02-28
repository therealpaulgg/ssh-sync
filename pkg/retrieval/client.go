package retrieval

import "github.com/therealpaulgg/ssh-sync/pkg/utils"

type Client struct {
	GetToken          func() (string, error)
	RetrieveMasterKey func() ([]byte, error)
}

func NewClient() Client {
	return Client{
		GetToken:          utils.GetToken,
		RetrieveMasterKey: utils.RetrieveMasterKey,
	}
}
