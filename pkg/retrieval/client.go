package retrieval

import "github.com/therealpaulgg/ssh-sync/pkg/utils"

type RetrievalClient struct {
	GetToken          func() (string, error)
	RetrieveMasterKey func() ([]byte, error)
}

func NewRetrievalClient() RetrievalClient {
	return RetrievalClient{
		GetToken:          utils.GetToken,
		RetrieveMasterKey: utils.RetrieveMasterKey,
	}
}
