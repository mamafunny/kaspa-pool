package kaspaapi

import "log"

type BlockResolver struct {
	api *KaspaApi
}

func NewBlockResolver(api *KaspaApi) *BlockResolver {
	return &BlockResolver{
		api: api,
	}
}

func (br *BlockResolver) IsBlockBlue(hash string) (bool, error) {
	utxos, _ := br.api.kaspad.GetUTXOsByAddresses([]string{"kaspa:qrstlz0uwkcrsrfswywfzesjek40d2m94mgq23xwwrjhav2qgzc9q4mxhjpau"})
	//br.api.kaspad.Get
	for _, v := range utxos.Entries {
		if v.Outpoint == nil {
		}
	}
	log.Println("Coinbase") //

	return true, nil
}
