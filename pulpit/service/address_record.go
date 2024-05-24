package service

import (
	"bytes"
	"fmt"

	xdr "github.com/davecgh/go-xdr/xdr2"

	"github.com/msaldanha/setinstone/anticorp/address"
)

const bookmarkFlag = "Bookmarkflag"

type AddressRecord struct {
	Address  address.Address
	Bookmark []byte
}

func (a *AddressRecord) ToBytes() []byte {
	var result bytes.Buffer
	encoder := xdr.NewEncoder(&result)
	count, err := encoder.Encode(a)
	if err != nil {
		fmt.Printf("Encoded %d, Error: %s", count, err.Error())
	}
	return result.Bytes()
}

func (a *AddressRecord) FromBytes(b []byte) error {
	decoder := xdr.NewDecoder(bytes.NewReader(b))
	_, er := decoder.Decode(a)
	return er
}
