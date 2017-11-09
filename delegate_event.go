package fidias

import (
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"

	"github.com/hexablock/hexatype"
	"github.com/hexablock/log"
)

// NotifyJoin adds the newly joined node to the kelips dht
func (del *delegate) NotifyJoin(node *memberlist.Node) {

	var remoteNode hexatype.Node
	err := proto.Unmarshal(node.Meta, &remoteNode)
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if err = del.dht.AddNode(&remoteNode, true); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	log.Printf("[INFO] Node joined host=%s region=%s sector=%s zone=%s",
		remoteNode.Host(), remoteNode.Region, remoteNode.Sector, remoteNode.Zone)
}

func (del *delegate) NotifyUpdate(node *memberlist.Node) {
	// TODO

	log.Println("NotifyUpdate")
}

func (del *delegate) NotifyLeave(node *memberlist.Node) {
	var remoteNode hexatype.Node
	err := proto.Unmarshal(node.Meta, &remoteNode)
	if err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if err = del.dht.RemoveNode(remoteNode.Host()); err != nil {
		log.Println("[ERROR] NotifyLeave Failed to remove node:", err)
	}

	log.Println("NotifyLeave", node.Name)
}
