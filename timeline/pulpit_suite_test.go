package timeline_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPulpit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pulpit Suite")
}
