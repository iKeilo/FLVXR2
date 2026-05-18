package socket

import (
	"github.com/go-gost/x/nftables"
)

// NftablesManagerInterface defines the interface for nftables manager operations.
type NftablesManagerInterface interface {
	AddRule(forwardID, nodeID int64, protocol string, port int, target string, speedLimit int) error
	UpdateRule(forwardID int64, protocol string, port int, target string, speedLimit int) error
	DeleteRule(forwardID int64, protocol string) error
	GetCounters() []nftables.CounterResult
	ResetCounters() error
}
