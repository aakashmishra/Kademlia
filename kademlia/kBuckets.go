package kademlia

import (
	"log"
	"strings"
	"fmt"
	"container/vector"
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

type KBucket struct {
	Contacts []Contact
}

type FindWrap struct {
	nodeId      ID
	contactChan chan *Contact
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
	// log.Println("prefixLen: ", prefixLen)

	bucket = &kbs.buckets[159-prefixLen]
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
		pingMessage := kbs.kadem.DoPing(contactsSlice[0].Host, contactsSlice[0].Port)
		log.Printf("pinged! message: %v", pingMessage)
		if strings.HasPrefix(pingMessage, "OK:") {
			log.Println("update - ping success")
			bucket.Contacts = append(contactsSlice[1:], contactsSlice[0])
		} else {
			log.Println("update - ping failed")
			bucket.Contacts = append(contactsSlice[1:], contact)
		}
	}
}
func (rec *ContactRecord) Less(other interface{}) bool {
  return rec.sortKey.Less(other.(*ContactRecord).sortKey);
}

func copyToVector(start int, end int ,array []Contacts, vec *vector.Vector, target ID) {
  for elt := start; elt < end; elt ++ {
    contact := 	array[elt];
    vec.Push(&ContactRecord{contact, contact.id.Xor(target)});
  }
}

func (table *KBuckets) FindClosest(target ID, count int) (ret *vector.Vector) {
  ret = new(vector.Vector).Resize(0, count);
 //find which bucket it belongs to
  bucket_num := target.Xor(table.node.id).PrefixLen();
 //load the required Bucket into the bucket
  bucket := table.Buckets[bucket_num];
 //copy all the bucket items and store it in a vector ret
  copyToVector(0, len(bucket), bucket, ret, target);
 //if at all the length of the vector is not satisfied then the other buckets are taken into consideration
  for i := 1; (bucket_num-i >= 0 || bucket_num+i < IDBits) && ret.Len() < count; i++ {
    if bucket_num - i >= 0 {
      bucket = table.buckets[bucket_num - i];
      copyToVector(0, len(bucket), bucket,ret, target);
    }
    if bucket_num + i < IDBits{
      bucket = table.buckets[bucket_num + i];
      copyToVector(0, len(bucket), bucket, ret, target);
    }
  }
  
  sort.Sort(ret);
  if ret.Len() > count {
    ret.Cut(count, ret.Len());
  }
  return;
}
