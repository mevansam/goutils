package network

import (
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/mevansam/goutils/logger"
	"github.com/mitchellh/go-homedir"
)

var (
	home string

	nullOut *os.File
)

func init() {

	var (
		err error
	)

	if home, err = homedir.Dir(); err != nil {
		logger.ErrorMessage("network.init(): Error determining home dir: %s", err.Error())
		panic(err)
	}
	if nullOut, err = os.Open(os.DevNull); err != nil {
		logger.ErrorMessage("network.init(): Error getting the null file output handle: %s", err.Error())
		panic(err)
	}
}

// given a device name prefix return the next available one
func GetNextAvailabeInterface(prefix string) (string, error) {

	var (
		err error

		devNamePattern *regexp.Regexp
		matches        [][]string

		ifaces   []net.Interface
		devIndex, maxIndex int
	)

	if devNamePattern, err = regexp.Compile("^" + prefix + "([0-9]+)$"); err != nil {
		return "", err
	}
	if ifaces, err = net.Interfaces(); err != nil {
		return "", err
	}
	maxIndex = 0
	for _, i := range ifaces {

		if matches = devNamePattern.FindAllStringSubmatch(i.Name, -1); matches != nil {
			if devIndex, err = strconv.Atoi(matches[0][1]); err != nil {
				continue
			}
			if maxIndex < devIndex {
				maxIndex = devIndex
			}
		}
	}
	return prefix + strconv.FormatInt(int64(maxIndex+1), 10), nil
}

func IncIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}