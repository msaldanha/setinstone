package keypair

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestKeypair(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keypair Suite")
}
