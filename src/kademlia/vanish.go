package kademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	mathrand "math/rand"
	"sss"
	"time"
	"log"
)

type VanashingDataObject struct {
	AccessKey  int64
	Ciphertext []byte
	NumberKeys byte
	Threshold  byte
}

func GenerateRandomCryptoKey() (ret []byte) {
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey() (accessKey int64) {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	accessKey = r.Int63()
	return
}

func CalculateSharedKeyLocations(accessKey int64, count int64) (ids []ID) {
	r := mathrand.New(mathrand.NewSource(accessKey))
	ids = make([]ID, count)
	for i := int64(0); i < count; i++ {
		for j := 0; j < IDBytes; j++ {
			ids[i][j] = uint8(r.Intn(256))
		}
	}
	return
}

func encrypt(key []byte, text []byte) (ciphertext []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return
}

func decrypt(key []byte, ciphertext []byte) (text []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext is not long enough")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext
}

func VanishData(kadem Kademlia, data []byte, numberKeys byte,
	threshold byte,vdoid ID) (vdo VanashingDataObject) {
	key := GenerateRandomCryptoKey()
	ciphertext := encrypt(key, data)
	keyPieces, _ := sss.Split(numberKeys, threshold, key)
	accessKey := GenerateRandomAccessKey()
	ids := CalculateSharedKeyLocations(accessKey, int64(numberKeys))

	for x := byte(1); x <= numberKeys; x++ {
		fmt.Println("key part: %s", x)
		fmt.Println("value part: %s", keyPieces[x])
		all := append([]byte{x}, keyPieces[x]...)
		fmt.Println("all to be stored: %s", all)

		// Problem!
		// We need a way to do later Find value and get these values back
		kadem.DoIterativeStore(ids[x-1], all)
	}

	vdo.AccessKey = accessKey
	vdo.Ciphertext = ciphertext
	vdo.NumberKeys = numberKeys
	vdo.Threshold = threshold
	vdo_push := new(Add_VDO)
	vdo_push.key = vdoid
	vdo_push.VDO = vdo
	log.Println(vdo_push)
	log.Println(len(kadem.store_VDO))
	kadem.store_VDO <- *vdo_push
	fmt.Println("Created VDO is %s", vdo)

	return vdo
}

func UnvanishData(kadem Kademlia, vdo VanashingDataObject) (data []byte) {

	ids := CalculateSharedKeyLocations(vdo.AccessKey, int64(vdo.NumberKeys))
	keyPieces := make(map[byte][]byte)
	keyPiecesFound := byte(0)
	for x := 1; x <= len(ids); x++ {
		// Problem!
		// We need a way to ask for the right values from the nodes
		value := kadem.DoIterativeFindValueval(ids[x-1])
		log.Println(value)
		if value != nil{
			keyPieces[byte(x)] = value 
			keyPiecesFound = keyPiecesFound + 1
		}
		if keyPiecesFound >= vdo.Threshold {
			break
		}
	}
	if keyPiecesFound >= vdo.Threshold{
		key := sss.Combine(keyPieces)
		data = decrypt(key, vdo.Ciphertext)
	}
	log.Println(data)
	return data
}
