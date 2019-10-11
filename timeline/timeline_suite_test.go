package timeline_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTimeline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Timeline Suite")
}
