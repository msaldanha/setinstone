package resolver

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRecord(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Record Suite")
}
