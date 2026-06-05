//go:build !linux

package socket

import (
	"errors"
	"fmt"
)

func (w *WebSocketReporter) handleCheckWireGuardSupport(data interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"ip":        false,
		"wg":        false,
		"nft":       false,
		"supported": false,
		"message":   "wireguard path is only supported on Linux nodes",
	}, nil
}

func (w *WebSocketReporter) handleApplyWireGuardPath(data interface{}) (map[string]interface{}, error) {
	return nil, errors.New("wireguard path is only supported on Linux nodes")
}

func (w *WebSocketReporter) handleRemoveWireGuardPath(data interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"removed": false}, errors.New("wireguard path is only supported on Linux nodes")
}

func (w *WebSocketReporter) handleGetWireGuardPathStatus(data interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"up": false}, fmt.Errorf("wireguard path is only supported on Linux nodes")
}

func (w *WebSocketReporter) handleProbeWireGuardPath(data interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": false}, fmt.Errorf("wireguard path is only supported on Linux nodes")
}

func (w *WebSocketReporter) handleApplyWGForwardRule(data interface{}) (map[string]interface{}, error) {
	return nil, errors.New("WG forward rule is only supported on Linux nodes")
}

func (w *WebSocketReporter) handleRemoveWGForwardRule(data interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"removed": false}, errors.New("WG forward rule is only supported on Linux nodes")
}

func (w *WebSocketReporter) handleGetWGForwardRuleStatus(data interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"success": false}, fmt.Errorf("WG forward rule is only supported on Linux nodes")
}
