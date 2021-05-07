package message

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMessage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Message Suite")
}
