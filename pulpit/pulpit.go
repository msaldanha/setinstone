package pulpit

import (
	"context"
	"encoding/hex"
	"fmt"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/msaldanha/setinstone/anticorp/address"
	"github.com/msaldanha/setinstone/anticorp/dag"
	"github.com/msaldanha/setinstone/anticorp/datastore"
	"github.com/msaldanha/setinstone/anticorp/err"
	"github.com/msaldanha/setinstone/anticorp/graph"
	"github.com/msaldanha/setinstone/anticorp/keyvaluestore"
	"github.com/msaldanha/setinstone/anticorp/resolver"
	"github.com/msaldanha/setinstone/anticorp/util"
	"github.com/msaldanha/setinstone/timeline"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type pulpitService struct {
	store     keyvaluestore.KeyValueStore
	timelines map[string]timeline.Timeline
	ds        datastore.DataStore
	ipfs      icore.CoreAPI
	logins    map[string]string
	resolver  resolver.Resolver
}

func newPulpitService(store keyvaluestore.KeyValueStore, ds datastore.DataStore,
	ipfs icore.CoreAPI, resolver resolver.Resolver) pulpitService {
	return pulpitService{
		store:     store,
		ds:        ds,
		ipfs:      ipfs,
		resolver:  resolver,
		timelines: map[string]timeline.Timeline{},
		logins:    map[string]string{},
	}
}

func (s pulpitService) createAddress(ctx context.Context, pass string) (string, error) {
	if pass == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	a, er := address.NewAddressWithKeys()
	if er != nil {
		return "", er
	}

	dbAddress := a.Clone()
	dbAddress.Keys.PrivateKey = hex.EncodeToString(util.Encrypt([]byte(dbAddress.Keys.PrivateKey), pass))
	ar := AddressRecord{
		Address:  *dbAddress,
		Bookmark: util.Encrypt([]byte(bookmarkFlag), pass),
	}

	er = s.store.Put(dbAddress.Address, ar.ToBytes())
	if er != nil {
		return "", er
	}

	s.logins[a.Address] = pass

	er = s.resolver.Manage(a)
	if er != nil {
		return "", er
	}

	return a.Address, nil
}

func (s pulpitService) deleteAddress(ctx context.Context, addr string) error {
	_, found, er := s.store.Get(addr)
	if er != nil {
		return er
	}
	if !found {
		return err.Error("addr not found in local storage")
	}
	er = s.store.Delete(addr)
	if er != nil {
		return er
	}
	return nil
}

func (s pulpitService) login(ctx context.Context, addr, password string) error {
	if addr == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	buf, found, er := s.store.Get(addr)
	if er != nil {
		return er
	}
	if !found {
		return err.Error("invalid addr or password")
	}

	ar := AddressRecord{}
	er = ar.FromBytes(buf)
	if er != nil {
		return er
	}

	privKey, er := hex.DecodeString(ar.Address.Keys.PrivateKey)
	if er != nil {
		return er
	}

	_, er = util.Decrypt(privKey, password)
	if er != nil {
		return err.Error("invalid addr or password")
	}

	s.logins[addr] = password
	return nil
}

func (s pulpitService) getRandomAddress(ctx context.Context) (*address.Address, error) {
	a, er := address.NewAddressWithKeys()
	if er != nil {
		return nil, er
	}
	return a, nil
}

func (s pulpitService) getMedia(ctx context.Context, id string) (io.Reader, error) {
	p := path.New(id)

	node, er := s.ipfs.Unixfs().Get(ctx, p)
	if er == context.DeadlineExceeded {
		return nil, fmt.Errorf("not found: %s", id)
	}
	if er != nil {
		return nil, er
	}
	f, ok := node.(files.File)
	if !ok {
		return nil, fmt.Errorf("not a file: %s", id)
	}
	return f, nil
}

func (s pulpitService) postMedia(ctx context.Context, f []string) []AddMediaResult {
	results := []AddMediaResult{}
	c := context.Background()
	for _, v := range f {
		id, er := s.addFile(c, v)
		if er != nil {
			results = append(results, AddMediaResult{
				File:  v,
				Id:    id,
				Error: er.Error(),
			})
		} else {
			results = append(results, AddMediaResult{
				File:  v,
				Id:    id,
				Error: "",
			})
		}

	}
	return results
}

func (s pulpitService) getAddresses(ctx context.Context) ([]*address.Address, error) {
	all, er := s.store.GetAll()
	if er != nil {
		return nil, er
	}
	addresses := []*address.Address{}
	for _, v := range all {
		ar := AddressRecord{}
		_ = ar.FromBytes(v)
		addresses = append(addresses, &ar.Address)
	}
	return addresses, nil
}

func (s pulpitService) getItems(ctx context.Context, addr, ns, keyRoot, connector, from, to string, count int) ([]interface{}, error) {
	if connector == "" {
		connector = "main"
	}

	tl, er := s.getTimeline(ns, addr)
	if er != nil {
		return nil, er
	}

	items, er := tl.GetFrom(ctx, keyRoot, connector, from, to, count)
	if er != nil && er != timeline.ErrNotFound {
		return nil, er
	}

	payload := make([]interface{}, 0, len(items))
	for _, item := range items {
		i, _ := item.AsInterface()
		payload = append(payload, i)
	}

	return payload, nil
}

func (s pulpitService) getItemByKey(ctx context.Context, addr, ns, key string) (interface{}, error) {
	tl, er := s.getTimeline(ns, addr)
	if er != nil {
		return nil, er
	}

	item, ok, er := tl.Get(ctx, key)
	if er != nil {
		return nil, er
	}

	if ok {
		if i, ok := item.AsInterface(); ok {
			return i, nil
		}
		return nil, fmt.Errorf("unable to covert item")
	}

	return nil, nil
}

func (s pulpitService) createItem(ctx context.Context, addr, ns, keyRoot, connector string, body AddItemRequest) (string, error) {
	if connector == "" {
		connector = "main"
	}

	tl, er := s.getTimeline(ns, addr)
	if er != nil {
		return "", er
	}

	key := ""
	switch body.Type {
	case timeline.TypePost:
		key, er = s.createPost(ctx, tl, body.PostItem, keyRoot, connector)
	case timeline.TypeReference:
		key, er = s.createReference(ctx, tl, body.ReferenceItem, keyRoot, connector)
	default:
		er = fmt.Errorf("unknown type %s", body.Type)
		return "", er
	}

	return key, er
}

func (s pulpitService) createPost(ctx context.Context, tl timeline.Timeline, postItem PostItem, keyRoot, connector string) (string, error) {
	if len(postItem.Connectors) == 0 {
		er := fmt.Errorf("reference types cannot be empty")
		return "", er
	}
	for _, v := range postItem.Connectors {
		if v == "" {
			er := fmt.Errorf("reference types cannot contain empty value")
			return "", er
		}
	}

	post, er := s.toTimelinePost(postItem)
	if er != nil {
		return "", er
	}
	key, er := tl.AppendPost(ctx, post, keyRoot, connector)
	if er != nil {
		return "", er
	}
	return key, nil
}

func (s pulpitService) createReference(ctx context.Context, tl timeline.Timeline, postItem ReferenceItem, keyRoot, connector string) (string, error) {
	if postItem.Target == "" {
		er := fmt.Errorf("target cannot be empty")
		return "", er
	}

	post := s.toTimelineReference(postItem)

	key, er := tl.AppendReference(ctx, post, keyRoot, connector)
	if er != nil {
		return "", er
	}
	return key, nil
}

func (s pulpitService) getTimeline(ns, addr string) (timeline.Timeline, error) {
	tl, found := s.timelines[ns+addr]
	if found {
		return tl, nil
	}

	pass := s.logins[addr]

	a, er := s.getAddress(addr, pass)
	if er != nil {
		return nil, er
	}

	tl = s.createTimeLine(ns, a)
	return tl, nil
}

func (s pulpitService) getAddress(addr, pass string) (*address.Address, error) {
	var a address.Address
	a = address.Address{Address: addr}
	if pass != "" {
		buf, found, _ := s.store.Get(addr)
		if found {
			ar := AddressRecord{}
			er := ar.FromBytes(buf)
			if er != nil {
				return nil, er
			}
			a = ar.Address
			privKey, er := hex.DecodeString(ar.Address.Keys.PrivateKey)
			if er != nil {
				return nil, er
			}
			pk, er := util.Decrypt(privKey, pass)
			if er != nil {
				return nil, er
			}
			a.Keys.PrivateKey = string(pk)
		}
	}
	return &a, nil
}

func (s *pulpitService) createTimeLine(ns string, a *address.Address) timeline.Timeline {
	if a.Keys != nil && a.Keys.PrivateKey != "" {
		_ = s.resolver.Manage(a)
	}
	ld := dag.NewDag(ns, s.ds, s.resolver)
	gr := graph.NewGraph(ld, a)
	tl := timeline.NewTimeline(gr)
	s.timelines[ns+a.Address] = tl
	return tl
}

func (s pulpitService) toTimelineReference(referenceItem ReferenceItem) timeline.ReferenceItem {
	return timeline.ReferenceItem{
		Reference: timeline.Reference{
			Target:    referenceItem.Target,
			Connector: referenceItem.Connector,
		},
		Base: timeline.Base{
			Type: timeline.TypeReference,
		},
	}
}
func (s pulpitService) toTimelinePost(postItem PostItem) (timeline.PostItem, error) {
	post := timeline.Post{}
	post.Part = postItem.Part
	post.Links = postItem.Links
	c := context.Background()
	for i, v := range postItem.Attachments {
		mimeType, er := getFileContentType(v)
		if er != nil {
			return timeline.PostItem{}, er
		}
		cid, er := s.addFile(c, v)
		if er != nil {
			return timeline.PostItem{}, er
		}
		post.Attachments = append(post.Attachments, timeline.PostPart{
			Seq:  i + 1,
			Name: filepath.Base(v),
			Part: timeline.Part{
				MimeType: mimeType,
				Encoding: "",
				Data:     "ipfs://" + cid,
			},
		})
	}
	mi := timeline.PostItem{
		Post: post,
		Base: timeline.Base{
			Type:       timeline.TypePost,
			Connectors: postItem.Connectors,
		},
	}
	return mi, nil
}

func (s pulpitService) addFile(ctx context.Context, name string) (string, error) {
	someFile, er := getUnixfsNode(name)
	if er != nil {
		return "", er
	}

	cidFile, er := s.ipfs.Unixfs().Add(ctx, someFile)
	if er != nil {
		return "", er
	}

	fmt.Printf("Added file to IPFS with CID %s\n", cidFile.String())
	return cidFile.String(), nil
}

func getFileContentType(path string) (string, error) {
	f, er := os.Open(path)
	if er != nil {
		return "", er
	}
	defer f.Close()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, er = f.Read(buffer)
	if er != nil {
		return "", er
	}

	// Use the net/http package's handy DetectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}
