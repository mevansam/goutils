package persistence_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/mevansam/goutils/logger"
)

func TestData(t *testing.T) {
	logger.Initialize()

	RegisterFailHandler(Fail)
	RunSpecs(t, "data persistence")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
