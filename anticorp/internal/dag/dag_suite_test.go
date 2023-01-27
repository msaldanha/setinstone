package dag_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDag(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dag Suite")
}
