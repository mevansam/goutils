package ux_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/mevansam/goutils/logger"
)

func TestUX(t *testing.T) {
	logger.Initialize()

	RegisterFailHandler(Fail)
	RunSpecs(t, "UX")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
