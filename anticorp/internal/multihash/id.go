package multihash

import (
	cid "github.com/ipfs/go-cid"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
)

type Id interface {
	String() string
	SetData([]byte) error
	Digest() ([]byte, error)
	Cid() cid.Cid
}

type id struct {
	pref cid.Prefix
	cid  cid.Cid
}

func NewId() Id {
	var pref cid.Prefix
	pref.Version = 0
	pref.Codec = uint64(mc.Cidv2)
	pref.MhType = mh.SHA2_256
	pref.MhLength = 32
	return &id{
		pref: pref,
	}
}

func NewIdFromString(v string) (Id, error) {
	c, er := cid.Decode(v)
	if er != nil {
		return nil, er
	}
	return &id{
		pref: c.Prefix(),
		cid:  c,
	}, nil
}

func (i *id) SetData(data []byte) error {
	c, er := i.pref.Sum(data)
	if er != nil {
		return er
	}
	i.cid = c
	return nil
}

func (i *id) String() string {
	if !i.cid.Defined() {
		return ""
	}
	return i.cid.String()
}

func (i *id) Digest() ([]byte, error) {
	if !i.cid.Defined() {
		return []byte{}, nil
	}
	dm, er := mh.Decode(i.cid.Hash())
	if er != nil {
		return []byte{}, er
	}

	return dm.Digest, nil
}

func (i *id) Cid() cid.Cid {
	return i.cid
}
