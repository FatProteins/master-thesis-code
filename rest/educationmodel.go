package rest

type KVPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Key struct {
	Key string `json:"key"`
}

type ActionTypeRequest struct {
	ActionType string `json:"actionType"`
}

type AllKVResponse struct {
	Pairs []KVPair `json:"pairs"`
}
