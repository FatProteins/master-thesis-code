package rest

type KVPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KeyDelete struct {
	Key string `json:"key"`
}

type ActionTypeRequest struct {
	ActionType string `json:"actionType"`
}

type KvWatchUpdate struct {
	ChangeLog []string `json:"changeLog"`
}

type KvGetUpdate struct {
	Pairs []KVPair `json:"pairs"`
}

type StepByStepRequest struct {
	Enable bool `json:"enable"`
}

type NodeStatusUpdate struct {
	MemberState  string `json:"memberState"`
	NodeId       string `json:"nodeId"`
	Leader       string `json:"leader"`
	Term         uint64 `json:"term"`
	Index        uint64 `json:"index"`
	AppliedIndex uint64 `json:"appliedIndex"`
	StatusError  bool   `json:"statusError"`
}

type StatusResponse struct {
	CurrentState   string `json:"currentState"`
	StepByStepMode bool   `json:"stepByStepMode"`
}

type LogUpdate struct {
	LogMessage string `json:"logMessage"`
}
