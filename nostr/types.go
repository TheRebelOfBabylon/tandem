package nostr

type Event struct {
	EventId   string `json:"id"`
	Pubkey    string
	CreatedAt uint32 `json:"created_at"`
	Kind      uint16
	Tags      [][]string
	Content   string
	Sig       string
}

type Filter struct {
	Ids     []string
	Authors []string
	Kinds   []uint16
	Tags    [][]string
	Since   uint64
	Until   uint64
	Limit   uint16
}

type Request struct {
	SubscriptionId string
	Filters        []Filter
}

type Close struct {
	SubscriptionId string
}
