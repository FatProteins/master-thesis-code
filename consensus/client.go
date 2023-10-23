package consensus

type ConsensusClient interface {
	GetAllKV(addKV func(string, string)) error
	StoreKV(key string, value string) error
}
