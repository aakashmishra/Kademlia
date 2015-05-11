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

type FindReq struct {
	key       ID
	valueChan chan []byte
}

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID      ID
	SelfContact Contact
	Kbs         *KBuckets
	HashTable   map[ID][]byte
	StoreChan   chan StoreRequest
	FindChan    chan FindReq
}

func NewKademlia(laddr string) *Kademlia {
	// TODO: Initialize other state here as you add functionality.
	kadem := new(Kademlia)
	kadem.NodeID = NewRandomID()
	kadem.Kbs = CreateKBuckets(k, kadem.NodeID, kadem)

	kadem.HashTable = make(map[ID][]byte)
	kadem.StoreChan = make(chan StoreRequest, 100)
	kadem.FindChan = make(chan FindReq, 100)

	go kadem.LoopOverMap()

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
		log.Println(contacts[i].NodeID.AsString())
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

func (k *Kademlia) LoopOverMap() {
	for {
		select {
		case storeReq := <-k.StoreChan:
			k.HashTable[storeReq.Key] = storeReq.Value
		case findReq := <-k.FindChan:
			findReq.valueChan <- k.HashTable[findReq.key]
		}

	}
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
	ping.Sender = k.SelfContact
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
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Printf("DialHTTP: ", err)
		return "ERR: Not able to connect"
	}

	sender := new(StoreRequest)
	receiver := new(StoreResult)
	sender.Sender = *contact
	sender.MsgID = NewRandomID()
	sender.Key = key
	sender.Value = value

	err = client.Call("KademliaCore.Store", sender, &receiver)
	if err != nil {
		log.Printf("Call: ", err)
		return "ERR: Store Failed"
	}

	return "OK:" + receiver.MsgID.AsString()
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) string {
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	log.Printf(peerStr)
	client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Printf("DialHTTP: ", err)
		return "ERR: Not able to connect"
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
	for i := 0; i < len(receive.Nodes); i++ {
		k.Kbs.Update(receive.Nodes[i])
		log.Printf(receive.Nodes[i].NodeID.AsString())
		// log.Printf("contact: %v", receive.Nodes[i])
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
		log.Printf("DialHTTP: ", err)
		return "ERR: Not able to connect"
	}

	send := new(FindValueRequest)
	receive := new(FindValueResult)
	send.Sender = *contact
	send.MsgID = NewRandomID()
	send.Key = searchKey

	err = client.Call("KademliaCore.FindValue", send, &receive)
	if err != nil {
		log.Printf("Call: ", err)
		return "ERR: Not implemented"
	}
	log.Println(string(receive.Value))
	for i := 0; i < len(receive.Nodes); i++ {
		k.Kbs.Update(receive.Nodes[i])
		log.Printf(receive.Nodes[i].NodeID.AsString())
	}
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	return "OK" + receive.MsgID.AsString()
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"

}


func (k *Kademlia) LocalFindValue(searchKey ID) string {
	findReq := FindReq{searchKey, make(chan []byte, 1)}
	k.FindChan <- findReq
	val := <-findReq.valueChan
	if val == nil {
		return "ERR: Not Found"
	} else {
		return "OK: " + string(val)
	}
}

func (k *Kademlia) LocalFindValueval(searchKey ID) []byte {
	findReq := FindReq{searchKey, make(chan []byte, 1)}
	k.FindChan <- findReq
	val := <-findReq.valueChan
	return val
}

func (k *Kademlia) DoFindNodeiter(contact *Contact, searchKey ID, list chan<- Contact,done chan <- int) string {
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	log.Printf(peerStr)
	client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Printf("DialHTTP: ", err)
		done <- 0
		return "ERR: Not able to connect"
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
		done <- 0
		return "ERR: Not implemented"
	}
	for i := 0; i < len(receive.Nodes); i++ {
		k.Kbs.Update(receive.Nodes[i])
		// log.Printf(receive.Nodes[i].NodeID.AsString())
		list <- receive.Nodes[i]
		// log.Printf("contact: %v", receive.Nodes[i])
		//log.Printf(receive.Nodes[i].NodeID.AsString())
	}
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	done <- len(receive.Nodes)
	return "OK" + receive.MsgID.AsString()

	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"

}

func (k *Kademlia) DoIterativeFindNode(id ID) string {
	top3 := k.Kbs.FindClosest(id,alpha)
	log.Println(top3)
	y := *top3
	check_count := 0
	log.Println(y)
	shortlist := make([]ContactRecord, 0)
	for{
		top20list := make([]ContactRecord, 0)
		log.Println("Round start")
		list:= make(chan Contact,60)
		done:= make(chan int,3)
		// count:= 3
    	if len(shortlist) == 0{
			for i := 0; i < len(*top3); i++ {
		// k.Kbs.Update(receive.Nodes[i])
				// log.Println("I am in here")
				// log.Println(*y[i].contact)
				cont := y[i].contact
				go k.DoFindNodeiter(cont,id,list,done)
		// log.Printf(cont.NodeID.AsString())
		}
   		}else{
   			if len(shortlist)<3{
   				y = shortlist
   			}else{
   				y = shortlist[:3]
  	 	}
   		for i := 0; i < len(y); i++ {
		// k.Kbs.Update(receive.Nodes[i])
   // 			log.Println("---------else")
			// log.Println(y[i].contact)
			cont := y[i].contact
			go k.DoFindNodeiter(cont,id,list,done)
		// log.Printf(cont.NodeID.AsString())
		}
   		}
    	// fmt.Println(len(list))
  //   if len(list) == 0{
		// 	break
		// }
    // con1 := <- list
    // log.Println("Second")
    // top20list = append(top20list,ContactRecord{&con1, con1.NodeID.Xor(id)})
    // fmt.Println(len(list))
   		sum := 0
   		for count1:=0;count1<3;count1++{
   			buffer := <- done 
   			sum = sum + buffer

   		}
   		log.Println(sum)
   		for i:= 0;i<sum;i++{
   			con := <- list
   			// log.Println("I am here inside list")
			duplicate := 0
			conta := ContactRecord{&con, con.NodeID.Xor(id)}
			// log.Println(conta)
			// log.Println(&con)
			// to avoid duplication of data
			for j:=0;j<len(top20list);j++{
				if conta.sortKey == top20list[j].sortKey{
					duplicate  = 1
					break
					}
				}
				if duplicate == 0{
					// log.Println("c")
					top20list = append(top20list,ContactRecord{&con, con.NodeID.Xor(id)})
					}
   		}
  //   	for {
  //   		if count == 0 {
  //   			break
  //  		 	}
		// 	select{
		// 		case con := <- list:
			// 		log.Println("I am here inside list")
			// 		duplicate := 0
			// 		conta := ContactRecord{&con, con.NodeID.Xor(id)}
			// 		log.Println(conta)
			// // log.Println(&con)
			// // to avoid duplication of data
			// 		for i:=0;i<len(top20list);i++{
			// 			if conta == top20list[i]{
			// 				duplicate  = 1
			// 			}
			// 		}
			// 		if duplicate == 0{
			// 			log.Println("c")
			// 			top20list = append(top20list,ContactRecord{&con, con.NodeID.Xor(id)})
			// 		}
		// 		case  <- done:
		// 			count--
		// 			log.Println(count)
		// 		if count == 0{
		// 				break
		// 		}

		// 	}
		// }
	log.Println(len(top20list))
		for i:=0;i<len(shortlist);i++{
			duplicate := 0
			// to avoid duplication of data
			for j:=0;j<len(top20list);j++{
				// compare1 :=*shortlist[i].contact
				// compare2 := *top20list[j].contact
				if  shortlist[i].sortKey==top20list[j].sortKey {
					duplicate  = 1
					break
				}
			}
			if duplicate == 0{
				top20list = append(top20list,shortlist[i])
			}
		
		}

	sortKey := func(p1,p2 *ContactRecord) bool {
		return p1.sortKey.Less(p2.sortKey)
	}
	By(sortKey).Sort(top20list)
	// NodeID := func(p1 Contact,p2 ID) bool {
	// 	return p1.NodeID.Less(p2)
	// }
	// By(NodeID).Sort(top20list)
    if len(top20list) > 20 {
		//ret.Cut(count, ret.Len());
		top20list = top20list[:20]
	}
	log.Println(top20list)
	check := 0

	for i:=0;i<len(shortlist);i++{
		for j:=0;j<len(top20list);j++{
			if shortlist[i].sortKey == top20list[j].sortKey{
				check  = check + 1
			}
		}
	}
	val := 0
	if len(top20list) < len(shortlist){
		val =len(top20list)
	}else{
		val =len(shortlist)
	}
	// log.Println(val)
	// log.Println(top20list)
	log.Println("*************")
    
	if check == val && check_count != 0{
		check_count ++
		log.Println("Round over")
		break
	}
	// log.Println(top20list)
	// log.Println("*************")
	// for i:= 0;i<len(top20list);i++{
	// 	log.Println(*top20list[i].contact)
	// }
		shortlist = top20list
		check_count ++
	log.Println("Round over")
	}
	log.Println("It took me these many iterations")
	log.Println(check_count)
	for i:= 0;i<len(shortlist);i++{
		cont := *shortlist[i].contact
		log.Println(cont.NodeID.AsString())
	}
	
	// // For project 2!
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
