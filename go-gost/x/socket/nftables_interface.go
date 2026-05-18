package socket

// NftablesCounterResult holds counter data from nftables rules.
type NftablesCounterResult struct {
	ForwardID int64  `json:"forward_id"`
	Protocol  string `json:"protocol"`
	Port      int    `json:"port"`
	Packets   uint64 `json:"packets"`
	Bytes     uint64 `json:"bytes"`
}

// NftablesManagerInterface defines the interface for nftables manager operations.
type NftablesManagerInterface interface {
	AddRule(forwardID, nodeID int64, protocol string, port int, target string, speedLimit int) error
	UpdateRule(forwardID int64, protocol string, port int, target string, speedLimit int) error
	DeleteRule(forwardID int64, protocol string) error
	GetCounters() []NftablesCounterResult
	ResetCounters() error
}
