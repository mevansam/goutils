package entry_test

import (
	"path"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/mevansam/goutils/logger"
)

var (
	workingDirectory string
)

func TestData(t *testing.T) {
	logger.Initialize()

	_, filename, _, _ := runtime.Caller(0)
	workingDirectory = path.Dir(filename)

	RegisterFailHandler(Fail)
	RunSpecs(t, "data entry")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
