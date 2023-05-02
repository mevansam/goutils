package network_test

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
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

func outputMatcher(buffer bytes.Buffer, matchers []*regexp.Regexp) (int, int) {
	
	numMatches := 0
	scanner := bufio.NewScanner(bytes.NewReader(buffer.Bytes()))

	for scanner.Scan() {
		line := scanner.Text()
		for i, m := range matchers {
			if m.MatchString(line) { 
				numMatches++
				matchers = append(matchers[:i], matchers[i+1:]...)
				break
			}
		}
		fmt.Printf("Match test count: %s <= %d\n", line, numMatches)
	}
	return numMatches, len(matchers)
}
