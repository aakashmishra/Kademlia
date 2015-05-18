package kademlia

// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import (
	"net"
	"strconv"
	"strings"
	// "log"
)

type KademliaCore struct {
	kademlia *Kademlia
}

// Host identification.
type Contact struct {
	NodeID ID
	Host   net.IP
	Port   uint16
}

func (contact Contact) AsString() string {
	return "(" + contact.NodeID.AsString() + " - " + contact.Host.String() + ":" + strconv.FormatUint(uint64(contact.Port), 10) + ")"
}
func ContactsListAsString(cList []Contact) string {
	resultStr := "{"
	for i := 0; i < len(cList); i++ {
		resultStr += cList[i].AsString()
		if i != len(cList)-1 {
			resultStr += " , "
		}
	}
	resultStr += "}"
	return resultStr
}

///////////////////////////////////////////////////////////////////////////////
// PING
///////////////////////////////////////////////////////////////////////////////
type PingMessage struct {
	Sender Contact
	MsgID  ID
}

type PongMessage struct {
	MsgID  ID
	Sender Contact
}

func (kc *KademliaCore) Ping(ping PingMessage, pong *PongMessage) error {
	// TODO: Finish implementation
	pong.MsgID = CopyID(ping.MsgID)
	pong.Sender = kc.kademlia.SelfContact
	kc.kademlia.Kbs.Update(ping.Sender)
	// Specify the sender
	// Update contact, etc
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// STORE
///////////////////////////////////////////////////////////////////////////////
type StoreRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
	Value  []byte
}

type StoreResult struct {
	MsgID ID
	Err   error
}

func (kc *KademliaCore) Store(req StoreRequest, res *StoreResult) error {
	// TODO: Implement.
	kc.kademlia.StoreChan <- req
	kc.kademlia.Kbs.Update(req.Sender)
	res.MsgID = NewRandomID()
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_NODE
///////////////////////////////////////////////////////////////////////////////
type FindNodeRequest struct {
	Sender Contact
	MsgID  ID
	NodeID ID
}

type FindNodeResult struct {
	MsgID ID
	Nodes []Contact
	Err   error
}

func (kc *KademliaCore) FindNode(req FindNodeRequest, res *FindNodeResult) error {
	// TODO: Implement.
	res.MsgID = NewRandomID()
	// log.Println(req.NodeID)
	// to find the k contacts closest to the NodeID
	//left error implementation
	//closest 20 contacts given
	lis := kc.kademlia.Kbs.FindClosest(req.NodeID, k)
	res.Nodes = make([]Contact, len(*lis))
	y := *lis
	for i := 0; i < len(*lis); i++ {
		res.Nodes[i] = *y[i].contact
	}

	return nil

}

///////////////////////////////////////////////////////////////////////////////
// FIND_VALUE
///////////////////////////////////////////////////////////////////////////////
type FindValueRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
}

// If Value is nil, it should be ignored, and Nodes means the same as in a
// FindNodeResult.
type FindValueResult struct {
	MsgID ID
	Value []byte
	Nodes []Contact
	Err   error
}

func (kc *KademliaCore) FindValue(req FindValueRequest, res *FindValueResult) error {
	// TODO: Implement.
	res.MsgID = NewRandomID()
	// to find the k contacts closest to the NodeID
	//left error implementation
	//closest 20 contacts given
	if strings.HasPrefix(kc.kademlia.LocalFindValue(req.Key), "ERR:") {
		lis := kc.kademlia.Kbs.FindClosest(req.Key, k)
		res.Nodes = make([]Contact, len(*lis))
		y := *lis
		for i := 0; i < len(*lis); i++ {
			res.Nodes[i] = *y[i].contact
		}
	} else {
		res.Value = kc.kademlia.LocalFindValueval(req.Key)

	}

	return nil
	// res.MsgID = RandomID()

	// if (kc.kademlia.key == req.Key){
	// 	res.Value == hash<kc.kademlia.key>
	// }else{
	// // to find the k contacts closest to the NodeID
	// //left error implementation
	// //closest 20 contacts given
	// lis := kc.kademlia.Kbs.FindClosest(req.NodeID,20)
	// res.Nodes := make([]Contact,lis.len())
	// for i := 0; i < lis.Len(); i++ {
	//      res.Nodes[i] = *lis.At(i).(*ContactRecord).node
	//    	}
	//  	}
	//  }

}
