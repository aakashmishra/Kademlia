package kademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	k     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID      ID
	SelfContact Contact
	Kbs         *KBuckets
}

func NewKademlia(laddr string) *Kademlia {
	// TODO: Initialize other state here as you add functionality.
	kadem := new(Kademlia)
	kadem.NodeID = NewRandomID()
	kadem.Kbs = CreateKBuckets(k, kadem.NodeID, kadem)

	// Set up RPC server
	// NOTE: KademliaCore is just a wrapper around Kademlia. This type includes
	// the RPC functions.
	rpc.Register(&KademliaCore{kadem})
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}
	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ := net.SplitHostPort(l.Addr().String())
	port_int, _ := strconv.Atoi(port)
	ipAddrStrings, err := net.LookupHost(hostname)
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}
	kadem.SelfContact = Contact{kadem.NodeID, host, uint16(port_int)}

	// TESTING!!!!!
	log.Printf("Testing Starts!!!!")
	numContacts := 20
	var contacts [20]Contact
	for i := 0; i < numContacts; i++ {
		var id ID
		id[0] = byte(i)
		contacts[i] = Contact{id, host, uint16(1609 + i)}
	}
	log.Printf("contacts %v", contacts)

	for i := 0; i < numContacts; i++ {
		kadem.Kbs.Update(contacts[i])
	}
	time.Sleep(1 * time.Second)

	// log.Printf("kadem.Kbs.buckets %v", kadem.Kbs.buckets)

	// for i := 0; i < 5; i++ {
	// 	kadem.Kbs.Update(contacts[i])
	// }
	// time.Sleep(1 * time.Second)

	// for i := 0; i < numContacts; i++ {
	// 	found, err := kadem.FindContact(contacts[i].NodeID)
	// 	log.Printf("found : %v err: %v", found, err)
	// }

	// contact_1 := Contact{NewRandomID(), host, uint16(1)}
	// contact_2 := Contact{NewRandomID(), host, uint16(2)}
	// contact_3 := Contact{NewRandomID(), host, uint16(3)}
	// contact_4 := Contact{NewRandomID(), host, uint16(4)}
	// contact_4.NodeID[19] = kadem.SelfContact.NodeID[19]
	// contact_4.NodeID[18] = kadem.SelfContact.NodeID[18]
	// contact_5 := Contact{NewRandomID(), host, uint16(port_int)}

	// log.Printf("contact_1 : %v", contact_1)
	// log.Printf("contact_2 : %v", contact_2)
	// log.Printf("contact_3 : %v", contact_3)
	// log.Printf("contact_4 : %v", contact_4)
	// log.Printf("contact_5 : %v", contact_5)

	// log.Printf("contact_1 distance: %v", kadem.Kbs.selfID.Xor(contact_1.NodeID))
	// log.Printf("contact_2 distance: %v", kadem.Kbs.selfID.Xor(contact_2.NodeID))
	// log.Printf("contact_3 distance: %v", kadem.Kbs.selfID.Xor(contact_3.NodeID))
	// log.Printf("contact_4 distance: %v", kadem.Kbs.selfID.Xor(contact_4.NodeID))

	// kadem.Kbs.Update(contact_1)
	// kadem.Kbs.Update(contact_2)
	// kadem.Kbs.Update(contact_3)
	// kadem.Kbs.Update(contact_4)

	// time.Sleep(1 * time.Second)

	// log.Printf("kadem.Kbs.buckets[158] : %v", kadem.Kbs.buckets[158])
	// log.Printf("kadem.Kbs.buckets[159] : %v", kadem.Kbs.buckets[159])

	// found, err := kadem.FindContact(contact_1.NodeID)
	// log.Printf("found : %v err: %v", found, err)
	// found, err = kadem.FindContact(contact_2.NodeID)
	// log.Printf("found : %v err: %v", found, err)
	// found, err = kadem.FindContact(contact_3.NodeID)
	// log.Printf("found : %v err: %v", found, err)
	// found, err = kadem.FindContact(contact_4.NodeID)
	// log.Printf("found : %v err: %v", found, err)
	// found, err = kadem.FindContact(contact_5.NodeID)
	// log.Printf("found : %v err: %v", found, err)

	// time.Sleep(30 * time.Second)
	// log.Printf("Testing Ends!!!!")

	return kadem
}

type NotFoundError struct {
	id  ID
	msg string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%x %s", e.id, e.msg)
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	// TODO: Search through contacts, find specified ID
	// Find contact with provided ID
	if nodeId == k.SelfContact.NodeID {
		return &k.SelfContact, nil
	}
	return k.Kbs.FindContact(nodeId)
}

// This is the function to perform the RPC
func (k *Kademlia) DoPing(host net.IP, port uint16) string {
	peerStr := host.String() + ":" + strconv.Itoa(int(port))
	client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Printf("DialHTTP failed on: %v", err)
		return "ERR: DialHTTP failed"
	}

	ping := new(PingMessage)
	ping.MsgID = NewRandomID()
	var pong PongMessage
	err = client.Call("KademliaCore.Ping", ping, &pong)
	if err != nil {
		log.Printf("Call: %v", err)
		return "ERR: Call failed"
	}

	log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
	log.Printf("pong msgID: %s\n", pong.MsgID.AsString())
	return "OK:" + pong.MsgID.AsString()
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	//return "ERR: Not implemented"
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) string {
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	return "ERR: Not implemented"
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) string {
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	log.Printf(peerStr)
	client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Printf("DialHTTP: ", err)
		return "ERR: Not implemented"
	}

	send := new(FindNodeRequest)
	receive := new(FindNodeResult)
	send.Sender = *contact
	send.MsgID = NewRandomID()
	send.NodeID = searchKey
	// log.Println("Dofind")
	// log.Println(send.NodeID)


	err = client.Call("KademliaCore.FindNode", send, &receive)
	if err != nil {
		log.Printf("Call: ", err)
		return "ERR: Not implemented"
	}
	for i := 0; i<len(receive.Nodes); i++ {
    	// log.Printf(receive.Nodes[i].NodeID.AsString())
    	log.Printf("contact: %v", receive.Nodes[i])
    	//log.Printf(receive.Nodes[i].NodeID.AsString())
    }
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	return "OK" + receive.MsgID.AsString()
}

func (k *Kademlia) DoFindValue(contact *Contact, searchKey ID) string {
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
		return "ERR: Not implemented"
	}

	send := new(FindValueRequest)
	receive := new(FindValueResult)
	send.Sender = *contact
	send.MsgID = NewRandomID()
	send.Key = searchKey

	err = client.Call("KademliaCore.FindValue", send, &receive)
	if err != nil {
		log.Fatal("Call: ", err)
		return "ERR: Not implemented"
	}
	for i := 0; i<len(receive.Nodes); i++ {
       log.Printf("contact: %v", receive.Nodes[i])
    }
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	return "OK" + receive.MsgID.AsString()
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	
}

func (k *Kademlia) LocalFindValue(searchKey ID) string {
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	return "ERR: Not implemented"
}

func (k *Kademlia) DoIterativeFindNode(id ID) string {
	// For project 2!
	return "ERR: Not implemented"
}
func (k *Kademlia) DoIterativeStore(key ID, value []byte) string {
	// For project 2!
	return "ERR: Not implemented"
}
func (k *Kademlia) DoIterativeFindValue(key ID) string {
	// For project 2!
	return "ERR: Not implemented"
}
