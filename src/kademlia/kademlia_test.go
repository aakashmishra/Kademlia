package kademlia

import (
    "testing"
    "net"
    "strconv"
    "log"
)


func StringToIpPort(laddr string) (ip net.IP, port uint16, err error){
    hostString, portString, err := net.SplitHostPort(laddr)
    if err != nil {
        return
    }
    ipStr, err := net.LookupHost(hostString)
    if err != nil {
        return
    }
    for i := 0; i < len(ipStr); i++ {
        ip = net.ParseIP(ipStr[i])
        if ip.To4() != nil {
            break
        }
    }
    portInt, err := strconv.Atoi(portString)
    port = uint16(portInt)
    return
}

func TestPing(t *testing.T) {
    instance := make([]string, 0)
    instance_kademlia := make([]*Kademlia,0)

    for i:=0;i<90;i++{
        port := 1609 + i
        conversion := strconv.Itoa(port)
        instance =  append(instance,conversion)
    }
    for i:= 0; i<len(instance);i++{
        buffer := NewKademlia("localhost:" + instance[i])
        instance_kademlia = append(instance_kademlia,buffer)
    }
    instance1 := NewKademlia("localhost:7890")
    instance2 := NewKademlia("localhost:7891")
    host2, port2, _ := StringToIpPort("localhost:7891")
    save := instance1.DoPing(host2, port2)
    log.Println("returned")
    log.Println(save)
    contact2, err := instance1.FindContact(instance2.NodeID)
    if err != nil {
        log.Println(err)
        t.Error("Instance 2's contact not found in Instance 1's contact list")
        return
    }
    contact1, err := instance2.FindContact(instance1.NodeID)
    if err != nil {
        t.Error("Instance 1's contact not found in Instance 2's contact list")
        return
    }
    if contact1.NodeID != instance1.NodeID {
        t.Error("Instance 1 ID incorrectly stored in Instance 2's contact list")
    }
    if contact2.NodeID != instance2.NodeID {
        t.Error("Instance 2 ID incorrectly stored in Instance 1's contact list")
    }
    return
}
