package kademlia

import (
	// "fmt"
	"log"
	"strings"
	// vector "container/vector"
	"sort"
)

type KBuckets struct {
	k          int
	selfID     ID
	kadem      *Kademlia
	updateChan chan Contact
	findChan   chan FindWrap
	buckets    [IDBits]KBucket
}
type ContactRecord struct {
	contact *Contact
	sortKey ID
}

type KBucket struct {
	Contacts []Contact
}

type FindWrap struct {
	nodeId      ID
	contactChan chan *Contact
}

type By func(p1, p2 *ContactRecord) bool

func (by By) Sort(ContactRecords []ContactRecord) {
	ps := &ContactRecordSorter{
		ContactRecords: ContactRecords,
		by:             by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

type ContactRecordSorter struct {
	ContactRecords []ContactRecord
	by             func(p1, p2 *ContactRecord) bool // Closure used in the Less method.
}

// type ByName struct{*[]ContactRecord}
func (s *ContactRecordSorter) Len() int {
	return len(s.ContactRecords)
}
func (s *ContactRecordSorter) Swap(i, j int) {
	s.ContactRecords[i], s.ContactRecords[j] = s.ContactRecords[j], s.ContactRecords[i]
}
func (s *ContactRecordSorter) Less(i, j int) bool {
	return s.by(&s.ContactRecords[i], &s.ContactRecords[j])
}

func CreateKBuckets(k int, selfID ID, kadem *Kademlia) (kbs *KBuckets) {
	var buckets [IDBits]KBucket
	kbs = &KBuckets{k, selfID, kadem, make(chan Contact, 100), make(chan FindWrap, 100), buckets}
	for _, bucket := range kbs.buckets {
		bucket.Contacts = make([]Contact, 0)
	}

	go kbs.loop()

	return
}

func (kbs *KBuckets) loop() {
	for {
		select {
		case contact := <-kbs.updateChan:
			kbs.update(contact)
		case findWrap := <-kbs.findChan:
			kbs.find(findWrap)
		}
	}
}

func (kbs *KBuckets) FindContact(nodeId ID) (*Contact, error) {
	findWrap := FindWrap{nodeId, make(chan *Contact, 1)}
	kbs.findChan <- findWrap

	var contact *Contact = <-findWrap.contactChan

	var err error = nil
	if contact == nil {
		err = &NotFoundError{nodeId, "Not found"}
	}

	return contact, err
}

func (kbs *KBuckets) find(findWrap FindWrap) {
	bucket, contactIndex := kbs.findBucketAndIndex(findWrap.nodeId)

	if contactIndex == -1 {
		findWrap.contactChan <- nil
	} else {
		findWrap.contactChan <- &bucket.Contacts[contactIndex]
	}
}

func (kbs *KBuckets) findBucketAndIndex(id ID) (bucket *KBucket, contactIndex int) {
	distance := kbs.selfID.Xor(id)
	prefixLen := distance.PrefixLen()

	bucket = &kbs.buckets[prefixLen]
	contactIndex = -1
	for index, otherContact := range bucket.Contacts {
		if id.Equals(otherContact.NodeID) {
			contactIndex = index
			break
		}
	}

	if false {
		// log.Println("selfId: 	", kbs.selfID)
		log.Println("id: 		", id)
		// log.Println("distance: 	", distance)
		// log.Println("prefixLen: ", prefixLen)
		log.Println("bucket:	", bucket)
	}

	return
}

func (kbs *KBuckets) Update(contact Contact) {
	kbs.updateChan <- contact
}

func (kbs *KBuckets) update(contact Contact) {
	bucket, contactIndex := kbs.findBucketAndIndex(contact.NodeID)
	// log.Printf("contactIndex: %v , %v", contactIndex)
	// log.Printf("bucket, contactIndex: %v , %v", bucket, contactIndex)
	contactsSlice := bucket.Contacts
	if contactIndex != -1 {
		log.Println("update - found in bucket")
		bucket.Contacts = append(contactsSlice[:contactIndex], append(contactsSlice[contactIndex+1:], contactsSlice[contactIndex])...)
	} else if len(contactsSlice) < kbs.k {
		log.Println("update - not found in bucket but there is room")
		bucket.Contacts = append(contactsSlice, contact)
	} else {
		_, pongMessage := kbs.kadem.DoPingNoUpdate(contactsSlice[0].Host, contactsSlice[0].Port)
		log.Printf("pinged! message: %v", pongMessage)
		if strings.HasPrefix(pongMessage, "OK:") {
			log.Println("update - ping success")
			bucket.Contacts = append(contactsSlice[1:], contactsSlice[0])
		} else {
			log.Println("update - ping failed")
			bucket.Contacts = append(contactsSlice[1:], contact)
		}
	}
}
func (rec *ContactRecord) Less(other interface{}) bool {
	return rec.sortKey.Less(other.(*ContactRecord).sortKey)
}

// func (s ByName) Less(i, j int) bool { return s.ContactRecord[i].sortKey.Less(s.ContactRecord[j].sortKey) }

func copyToVector(start int, end int, array []Contact, ret *[]ContactRecord, target ID) {
	//ret := make([]ContactRecord, 0)
	//log.Println(target)
	for elt := start; elt < end; elt++ {
		contact := array[elt]
		// log.Println(contact.NodeID.AsString())
		// log.Println(contact.NodeID.Xor(target))
		*ret = append(*ret, ContactRecord{&contact, contact.NodeID.Xor(target)})
		// vec.Push(&ContactRecord{contact, contact.id.Xor(target)});
	}

}

func (table *KBuckets) FindClosest(target ID, count int) *[]ContactRecord {
	ret := make([]ContactRecord, 0)
	// log.Println(target)
	//find which bucket it belongs to
	bucket_num := target.Xor(table.selfID).PrefixLen()
	//load the required Bucket into the bucket
	bucket := table.buckets[bucket_num]
	//copy all the bucket items and store it in a vector ret
	copyToVector(0, len(bucket.Contacts), bucket.Contacts, &ret, target)
	//if at all the length of the vector is not satisfied then the other buckets are taken into consideration
	for i := 1; (bucket_num-i >= 0 || bucket_num+i < IDBits) && len(ret) < count; i++ {
		if bucket_num-i >= 0 {
			bucket = table.buckets[bucket_num-i]
			copyToVector(0, len(bucket.Contacts), bucket.Contacts, &ret, target)
		}
		if bucket_num+i < IDBits {
			bucket = table.buckets[bucket_num+i]
			copyToVector(0, len(bucket.Contacts), bucket.Contacts, &ret, target)
		}
	}

	//sort.Sort(ret)

	sortKey := func(p1, p2 *ContactRecord) bool {
		return p1.sortKey.Less(p2.sortKey)
	}
	By(sortKey).Sort(ret)

	//sort.Sort({ret})
	if len(ret) > count {
		//ret.Cut(count, ret.Len());
		ret = ret[:count]
	}
	return &ret
}
