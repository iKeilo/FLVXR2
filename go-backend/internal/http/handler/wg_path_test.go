package handler

import (
	"reflect"
	"testing"

	"go-backend/internal/store/model"
)

func TestAllowedWGPathIPsForThreeNodeChain(t *testing.T) {
	segments := []model.PathSegment{
		{Sequence: 1, FromNodeID: 1, ToNodeID: 2},
		{Sequence: 2, FromNodeID: 2, ToNodeID: 3},
	}
	order := orderedWGPathNodeIDs(segments)
	index := map[int64]int{}
	for i, nodeID := range order {
		index[nodeID] = i
	}
	addresses := map[int64][]string{
		1: {"10.88.1.1/32"},
		2: {"10.88.1.2/32", "10.88.1.3/32"},
		3: {"10.88.1.4/32"},
	}

	gotForward := allowedWGPathIPs(order, index, addresses, 2, 1)
	wantForward := []string{"10.88.1.2/32", "10.88.1.3/32", "10.88.1.4/32"}
	if !reflect.DeepEqual(gotForward, wantForward) {
		t.Fatalf("forward allowed IPs = %#v, want %#v", gotForward, wantForward)
	}

	gotReverse := allowedWGPathIPs(order, index, addresses, 2, -1)
	wantReverse := []string{"10.88.1.1/32", "10.88.1.2/32", "10.88.1.3/32"}
	if !reflect.DeepEqual(gotReverse, wantReverse) {
		t.Fatalf("reverse allowed IPs = %#v, want %#v", gotReverse, wantReverse)
	}
}
