package pulpit

import (
	"context"
	"fmt"
	"github.com/msaldanha/setinstone/timeline"
	"net/http"
	"os"
	"path/filepath"
)

func (s server) toTimelineMessage(req AddMessageRequest) (timeline.Message, error) {
	msg := timeline.Message{}
	msg.Body = req.Body
	msg.Links = req.Links
	for i, v := range req.Attachments {
		mimeType, er := getFileContentType(v)
		if er != nil {
			return msg, er
		}
		cid, er := s.addFile(v)
		if er != nil {
			return msg, er
		}
		msg.Attachments = append(msg.Attachments, timeline.MessagePart{
			Seq:      i + 1,
			Name:     filepath.Base(v),
			MimeType: mimeType,
			Encoding: "",
			Data:     "ipfs://" + cid,
		})
	}
	return msg, nil
}

func (s server) addFile(name string) (string, error) {
	someFile, er := getUnixfsNode(name)
	if er != nil {
		return "", er
	}

	ctx := context.Background()
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

	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}