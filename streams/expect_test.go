package streams_test

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/mevansam/goutils/streams"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Expect Stream Interceptor", func() {

	var (
		err error

		outputBuffer strings.Builder
	)

	// send data to writer in chunks of given size
	writeData := func(w io.Writer, d []byte, s int) {

		var (
			j, k, l int
		)

		l = len(d)
		for i := 0; i < l; {
			j = i + s
			if j > l {
				j = l
			}
			k, err = w.Write(d[i:j])
			Expect(err).NotTo(HaveOccurred())
			i = i + k
		}
	}

	BeforeEach(func() {
		outputBuffer.Reset()
	})

	Context("send commands", func() {

		FIt("receives commands from expect stream", func() {

			// pipe to send data from client
			stInSender, _ := io.Pipe()

			es, stInReciever, stOutReciever := streams.NewExpectStream(stInSender, os.Stdout /*&outputBuffer*/)
			defer func() {
				stInReciever.Close()
				stOutReciever.Close()
			}()

			es.SetBufferSize(32)
			es.AddMultiLineExpect(
				`^Welcome to Ubuntu`,
				`^bastion-admin@cbs-test:\~\$`,
				"sudo su -\n",
				true,
			)
			es.AddExpect(
				`password for bastion-admin:`,
				"P@ssw0rd!\n",
				true,
			)
			es.AddExpect(
				`root@cbs-test:\~\#`,
				"ls -al /usr\n",
				true,
			)
			es.Start()

			// reader from which data sent to receiver can be retrieved
			recieverData := bufio.NewScanner(stInReciever)

			writeData(stOutReciever, []byte(testRecieverWelcome), 32)
			recieverData.Scan()
			command := recieverData.Text()
			Expect(command).To(Equal("sudo su -"))
			writeData(stOutReciever, []byte(command+"\n"), 32)

			writeData(stOutReciever, []byte(testRecieverSudoPassword), 32)
			recieverData.Scan()
			command = recieverData.Text()
			Expect(command).To(Equal("P@ssw0rd!"))
			writeData(stOutReciever, []byte(command+"\n"), 32)

			writeData(stOutReciever, []byte(testRecieverSudoPrompt), 32)
			recieverData.Scan()
			command = recieverData.Text()
			Expect(command).To(Equal("ls -al /usr"))
			writeData(stOutReciever, []byte(testRecieverListOutput), 32)
			writeData(stOutReciever, []byte(command+"\n"), 32)
		})
	})
})

const testRecieverWelcome = `Welcome to Ubuntu 18.04.3 LTS (GNU/Linux 5.0.0-1028-gcp x86_64)

* Documentation:  https://help.ubuntu.com
* Management:     https://landscape.canonical.com
* Support:        https://ubuntu.com/advantage

 System information as of Fri Jan 10 19:09:11 UTC 2020

 System load:                    0.84
 Usage of /:                     7.5% of 48.29GB
 Memory usage:                   9%
 Swap usage:                     0%
 Processes:                      344
 Users logged in:                0
 IP address for ens0:            192.168.0.1

* Overheard at KubeCon: "microk8s.status just blew my mind".

		https://microk8s.io/docs/commands#microk8s.status

* Canonical Livepatch is available for installation.
	- Reduce system reboots and improve kernel security. Activate at:
		https://ubuntu.com/livepatch

47 packages can be updated.
1 update is a security update.


Last login: Fri Jan 10 09:23:26 2020 from 94.202.78.17
bastion-admin@cbs-test:~$ `

const testRecieverSudoPassword = `[sudo] password for bastion-admin: `
const testRecieverSudoPrompt = `root@cbs-test:~# `
const testRecieverListOutput = `total 76
drwxr-xr-x  11 root root  4096 Nov 13 15:53 .
drwxr-xr-x  24 root root  4096 Jan 10 22:08 ..
drwxr-xr-x   2 root root 36864 Dec 18 09:19 bin
drwxr-xr-x   2 root root  4096 Apr 24  2018 games
drwxr-xr-x  43 root root  4096 Nov 13 15:51 include
drwxr-xr-x  79 root root  4096 Dec 18 09:17 lib
drwxr-xr-x   3 root root  4096 Nov 13 15:53 libexec
drwxr-xr-x  10 root root  4096 Nov 13 04:33 local
drwxr-xr-x   2 root root  4096 Dec 18 09:18 sbin
drwxr-xr-x 137 root root  4096 Dec 18 09:17 share
drwxr-xr-x   9 root root  4096 Jan  9 06:03 src
`
