package network_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mevansam/goutils/logger"
	"github.com/onsi/gomega/gexec"
)

func TestNetwork(t *testing.T) {
	logger.Initialize()

	RegisterFailHandler(Fail)
	RunSpecs(t, "network")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
