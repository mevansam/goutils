package forms_test

import (
	"path"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mevansam/goutils/logger"
	"github.com/onsi/gomega/gexec"
)

var (
	workingDirectory string
)

func TestData(t *testing.T) {
	logger.Initialize()

	_, filename, _, _ := runtime.Caller(0)
	workingDirectory = path.Dir(filename)

	RegisterFailHandler(Fail)
	RunSpecs(t, "forms")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
