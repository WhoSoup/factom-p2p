package p2p

import (
	"fmt"
	"testing"
)

func TestNewPartialPeerView(t *testing.T) {
	ppv := NewPartialPeerView()
	ppv.view["test"] = true

	fmt.Println(ppv.view)
}
