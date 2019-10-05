package dmap_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Map Suite")
}
