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
	"strings"
	// "time"
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

type active struct{
	id ID
	val int
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
	s := rpc.NewServer() // Create a new RPC server
	s.Register(&KademliaCore{kadem})
	_, port, _ := net.SplitHostPort(laddr)
	s.HandleHTTP(rpc.DefaultRPCPath+port, rpc.DefaultDebugPath+port)
	// rpc.Register(&KademliaCore{kadem})
	// rpc.HandleHTTP()
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
	// log.Printf("Testing Starts!!!!")
	// numContacts := 20
	// var contacts [20]Contact
	// for i := 0; i < numContacts; i++ {
	// 	var id ID
	// 	id[0] = byte(i)
	// 	contacts[i] = Contact{id, host, uint16(1609 + i)}
	// }
	// log.Printf("contacts %v", contacts)

	// for i := 0; i < numContacts; i++ {
	// 	kadem.Kbs.Update(contacts[i])
	// 	log.Println(contacts[i].NodeID.AsString())
	// }
	// time.Sleep(1 * time.Second)

	// log.Printf("kadem.Kbs.buckets %v", kadem.Kbs.buckets)

	// for i := 0; i < 5; i++ {
	// 	kadem.Kbs.Update(contacts[i])
	// }
	// time.Sleep(1 * time.Second)

	// log.Printf("kadem.Kbs.buckets %v", kadem.Kbs.buckets)

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
	contact, message := k.DoPingNoUpdate(host, port)
	mess := ""
	if strings.HasPrefix(message, "OK:") && contact != nil {
		log.Println("IN HERE")
		number := k.Kbs.Update(*contact)
		mess = strconv.Itoa(int(number))
	}
	
	return (mess + message)
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
}

func (k *Kademlia) DoPingNoUpdate(host net.IP, port uint16) (*Contact, string) {
	port_str := strconv.Itoa(int(port))
	peerStr := host.String() + ":" + strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", peerStr,rpc.DefaultRPCPath+port_str)
	// peerStr := host.String() + ":" + strconv.Itoa(int(port))
	// client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Printf("DialHTTP failed on: %v", err)
		return nil, "ERR: DialHTTP failed"
	}

	ping := new(PingMessage)
	ping.MsgID = NewRandomID()
	ping.Sender = k.SelfContact
	var pong PongMessage
	err = client.Call("KademliaCore.Ping", ping, &pong)
	if err != nil {
		log.Printf("Call: %v", err)
		return nil, "ERR: Call failed"
	}

	log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
	log.Printf("pong msgID: %s , sender: %s %s:%d \n", pong.MsgID.AsString(), pong.Sender.NodeID.AsString(), pong.Sender.Host, pong.Sender.Port)
	return &pong.Sender, "OK:" + pong.MsgID.AsString()
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) string {
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	port_str := strconv.Itoa(int(contact.Port))
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", peerStr,rpc.DefaultRPCPath+port_str)
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
	port_str := strconv.Itoa(int(contact.Port))
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", peerStr,rpc.DefaultRPCPath+port_str)
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

func (k *Kademlia) DoFindNodeiter(contact *Contact, searchKey ID, list chan<- Contact, done chan<- int,active_chan chan<- active) string {
	port_str := strconv.Itoa(int(contact.Port))
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", peerStr,rpc.DefaultRPCPath+port_str)
	activeval := active{}
	log.Printf(peerStr)
	conta := *contact
	if err != nil {
		log.Printf("DialHTTP: ", err)
		done <- 0
		activeval.id = conta.NodeID
		activeval.val = 2
		active_chan <- activeval
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
		activeval.id = conta.NodeID
		activeval.val = 2
		active_chan <- activeval
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
	activeval.id = conta.NodeID
	activeval.val = 1
	active_chan <- activeval

	return "OK" + receive.MsgID.AsString()

	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"

}
func (k *Kademlia) DoIterativeFindNode(id ID) string{
	contacts_list := internalDoIterativeFindNode(id)
	for i := 0; i < len(contacts_list); i++ {
		cont := contacts_list
		return_contact = append(return_contact,cont)
		log.Println(cont.NodeID.AsString())
	}
	return 
}
func (k *Kademlia) internalDoIterativeFindNode(id ID) []contact {
	active_map := make(map[ID]int)
	top3 := k.Kbs.FindClosest(id, alpha)
	log.Println(top3)
	y := *top3
	check_count := 0
	log.Println(y)
	shortlist := make([]ContactRecord, 0)
	for {
		top20list := make([]ContactRecord, 0)
		log.Println("Round start")
		list := make(chan Contact, 60)
		done := make(chan int, 3)
		active_chan := make(chan active, 3)
		tally := len(y)

		for i := 0; i < len(y); i++ {
				top20list = append(top20list,y[i])
				cont := y[i].contact
				go k.DoFindNodeiter(cont, id, list, done,active_chan)
			}
	
		sum := 0
		for count1 := 0; count1 < tally; count1++ {
			buffer := <-done
			sum = sum + buffer

		}
		for count1 := 0; count1 < tally; count1++  {
			buffer := <-active_chan
			active_map[buffer.id] = buffer.val
			log.Println("####map####")
			log.Println(buffer.id)
			log.Println(active_map[buffer.id])
			log.Println("####mapend####")
		}
		log.Println(len(active_map))
		for i:=0;i<len(top20list);i++{
			cont := y[i].contact
			if active_map[cont.NodeID] == 2{
				top20list =append(top20list[:i],top20list[i+1:]...)
			}

		}
		log.Println("sum all")
		log.Println(sum)
		for i := 0; i < sum; i++ {
			con := <-list
			duplicate := 0
			conta := ContactRecord{&con, con.NodeID.Xor(id)}
			log.Println(i)
			// to avoid duplication of data
			for j := 0; j < len(top20list); j++ {
				if conta.sortKey == top20list[j].sortKey {
					duplicate = 1
					break
				}
			}
			if duplicate == 0 {
				top20list = append(top20list, ContactRecord{&con, con.NodeID.Xor(id)})
			}
		}

	
		y = y[:0]
		
		log.Println(len(top20list))
		for i := 0; i < len(shortlist); i++ {
			duplicate := 0
			// to avoid duplication of data
			for j := 0; j < len(top20list); j++ {
				if shortlist[i].sortKey == top20list[j].sortKey {
					duplicate = 1
					break
				}
			}
			if duplicate == 0 {
				top20list = append(top20list, shortlist[i])
			}

		}

		sortKey := func(p1, p2 *ContactRecord) bool {
			return p1.sortKey.Less(p2.sortKey)
		}
		By(sortKey).Sort(top20list)
		top60list := top20list
		if len(top20list) > 20 {
			top20list = top20list[:20]
		}
		log.Println(top20list)
		check := 0

		for i := 0; i < len(shortlist); i++ {
			for j := 0; j < len(top20list); j++ {
				if shortlist[i].sortKey == top20list[j].sortKey {
					check = check + 1
				}
			}
		}
		check_active := 0
		check_remove := 0
		for j := 0; j < len(top20list); j++ {
			check_contact := *top20list[j].contact
			if active_map[check_contact.NodeID] == 1{
				check_active = check_active + 1
			}
			if active_map[check_contact.NodeID] == 2{
				top20list = append(top20list[:j],top20list[j+1:]...)
				check_remove = check_remove + 1
			}
		}
		if len(top60list)>20 && check_remove > 0{
			for i:=20;i<len(top60list);i++{
				top20list = append(top20list,top60list[i])
				if len(top20list) == 20{
					break
				}
			}
		}


		log.Println("******active_check**************")
		log.Println(check_active)
		val := 0
		if len(top20list) < len(shortlist) {
			val = len(top20list)
		} else {
			val = len(shortlist)
		}

		log.Println("*************")
		if check_active == len(top20list){
			check_count++
			shortlist = top20list
			log.Println("Round over due to all active nodes")
			break
		}
		check_close := 0
		if check_count != 0{
		closestnodenew := *top20list[0].contact
		closestnode := *shortlist[0].contact
		if (closestnodenew.NodeID == closestnode.NodeID){
			check_close = 1
			for i:= 1;i<len(top20list);i++{
				check_contact := *top20list[i].contact
				if active_map[check_contact.NodeID] != 1{
					y = append(y,top20list[i])
					}
				if len(y) == 3{
					break
				}
			}
		}
		if check == val && check_count != 0 {
			check_count++
			log.Println(len(active_map))
			shortlist = top20list
			log.Println("Round over due to shortlist being unchanged")
			// break
		}
		}

		shortlist = top20list
		check_count++
		if check_close == 0{
		count_inactive := 0
		for i:= 0; i<len(shortlist);i++{
				check_contact := *top20list[i].contact
				if active_map[check_contact.NodeID] == 0{
					y = append(y,top20list[i])
					count_inactive++
				}
			}
		if (count_inactive == 0){
			break
		}
		}

		log.Println("Round over")
	}
	log.Println("It took me these many iterations")
	log.Println(check_count)
	return_contact := make([]Contact, 0)
	for i := 0; i < len(shortlist); i++ {
		cont := *shortlist[i].contact
		return_contact = append(return_contact,cont)
		log.Println(cont.NodeID.AsString())
	}

	// // For project 2!
	return return_contact
}


func (k *Kademlia) DoIterativeStore(key ID, value []byte) string {
	contacts := k.internalDoIterativeFindNode(key)
	for i := 0; i < len(contacts); i++ {
		k.DoStore(&contacts[i], key, value)
	}

	return contacts[len(contacts)-1].NodeID.AsString()
}


func (k *Kademlia) DoIterativeFindValue(key ID) string {
	// For project 2!
	if strings.HasPrefix(k.LocalFindValue(key),"ERR:"){
			top3 := k.Kbs.FindClosest(key, alpha)
			log.Println(top3)
			y := *top3
			check_count := 0
			log.Println(y)
			shortlist := make([]ContactRecord, 0)
			for {
				top20list := make([]ContactRecord, 0)
				log.Println("Round start")
				list := make(chan Contact, 60)
				done := make(chan int, 3)
				valu := make(chan string,0)
				retValue := ""
				retBool := false
				// count:= 3
				if len(shortlist) == 0 {
					retBool = false
					for i := 0; i < len(*top3); i++ {
						// k.Kbs.Update(receive.Nodes[i])
						// log.Println("I am in here")
						// log.Println(*y[i].contact)
						cont := y[i].contact
						go k.DoFindValueiter(cont, key, list, done, valu)
							// log.Printf(cont.NodeID.AsString())
						retValue = <- valu
						if(retValue != ""){
							retBool = true
							//TODO: store value at closet node
							break
						}

					}
				if(retBool){
					return retValue
				}

				} else {
					if len(shortlist) < 3 {
						y = shortlist
					} else {
						y = shortlist[:3]
					}
					retBool = false
					for i := 0; i < len(y); i++ {
						// k.Kbs.Update(receive.Nodes[i])
						// 			log.Println("---------else")
						// log.Println(y[i].contact)
						cont := y[i].contact
						go k.DoFindValueiter(cont, key, list, done, valu)
							// log.Printf(cont.NodeID.AsString())
						retValue = <- valu
						if(retValue != ""){
							retBool = true
							//TODO: store value at closet node
							break
						}
					}
					if(retBool){
					return retValue
				}

				}
				
				sum := 0
				for count1 := 0; count1 < 3; count1++ {
					buffer := <-done
					sum = sum + buffer

				}
				log.Println(sum)
				for i := 0; i < sum; i++ {
					con := <-list
					// log.Println("I am here inside list")
					duplicate := 0
					conta := ContactRecord{&con, con.NodeID.Xor(key)}
					
					for j := 0; j < len(top20list); j++ {
						if conta.sortKey == top20list[j].sortKey {
							duplicate = 1
							break
						}
					}
					if duplicate == 0 {
						// log.Println("c")
						top20list = append(top20list, ContactRecord{&con, con.NodeID.Xor(key)})
					}
				}
				
				log.Println(len(top20list))
				for i := 0; i < len(shortlist); i++ {
					duplicate := 0
					// to avoid duplication of data
					for j := 0; j < len(top20list); j++ {
						// compare1 :=*shortlist[i].contact
						// compare2 := *top20list[j].contact
						if shortlist[i].sortKey == top20list[j].sortKey {
							duplicate = 1
							break
						}
					}
					if duplicate == 0 {
						top20list = append(top20list, shortlist[i])
					}

				}

				sortKey := func(p1, p2 *ContactRecord) bool {
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

				for i := 0; i < len(shortlist); i++ {
					for j := 0; j < len(top20list); j++ {
						if shortlist[i].sortKey == top20list[j].sortKey {
							check = check + 1
						}
					}
				}
				val := 0
				if len(top20list) < len(shortlist) {
					val = len(top20list)
				} else {
					val = len(shortlist)
				}
				// log.Println(val)
				// log.Println(top20list)
				log.Println("*************")

				if check == val && check_count != 0 {
					check_count++
					log.Println("Round over")
					break
				}
				// log.Println(top20list)
				// log.Println("*************")
				// for i:= 0;i<len(top20list);i++{
				// 	log.Println(*top20list[i].contact)
				// }
				shortlist = top20list
				check_count++
				log.Println("Round over")
			}
			log.Println("It took me these many iterations")
			log.Println(check_count)
			for i := 0; i < len(shortlist); i++ {
				cont := *shortlist[i].contact
				log.Println(cont.NodeID.AsString())
			}
	}else{
		retV := string(k.LocalFindValueval(key))
		return retV
	}

		
	


	return "ERR: Not implemented"
}


func (k *Kademlia) DoFindValueiter(contact *Contact, searchKey ID, list chan<- Contact, done chan<- int, valu chan<- string) string {
	peerStr := contact.Host.String() + ":" + strconv.Itoa(int(contact.Port))
	log.Printf(peerStr)
	client, err := rpc.DialHTTP("tcp", peerStr)
	if err != nil {
		log.Printf("DialHTTP: ", err)
		done <- 0
		valu <- ""
		return "ERR: Not able to connect"
	}
	send := new(FindValueRequest)
	receive := new(FindValueResult)
	send.Sender = *contact
	send.MsgID = NewRandomID()
	send.Key = searchKey
	retValue := ""
	// log.Println("Dofind")
	// log.Println(send.NodeID)

err = client.Call("KademliaCore.FindValue", send, &receive)
	if err != nil {
		log.Printf("Call: ", err)
		done <- 0
		valu <- ""
		return "ERR: Not implemented" 
	}
	log.Println(string(receive.Value))
	retValue = string(receive.Value)
	valu <- retValue
	for i := 0; i < len(receive.Nodes); i++ {
		k.Kbs.Update(receive.Nodes[i])
		log.Printf(receive.Nodes[i].NodeID.AsString())
		list <- receive.Nodes[i]
    }

	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	done <- len(receive.Nodes)
	return "OK" + receive.MsgID.AsString()

	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"

}
