package utils_test

import (
	"bytes"

	"github.com/mevansam/goutils/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("bytes utils tests", func() {

	Context("writes to buffer", func() {

		var (
			err    error
			output bytes.Buffer
		)

		It("writes at random position", func() {

			writeAtBuffer := utils.NewWriteAtBuffer(&output)

			writeAtBuffer.WriteAt([]byte("abcd"), 10)
			writeAtBuffer.WriteAt([]byte("56789"), 5)
			Expect(output.Len()).To(Equal(0))
			writeAtBuffer.WriteAt([]byte("01234"), 0)
			writeAtBuffer.WriteAt([]byte("fghij"), 15)
			Expect(output.String()).To(Equal("0123456789abcd"))
			writeAtBuffer.WriteAt([]byte("mno"), 22)
			writeAtBuffer.WriteAt([]byte("e"), 14)
			Expect(output.String()).To(Equal("0123456789abcdefghij"))
			writeAtBuffer.WriteAt([]byte("qrstu"), 25)
			writeAtBuffer.WriteAt([]byte("xyz"), 32)
			writeAtBuffer.WriteAt([]byte("kl"), 20)
			Expect(output.String()).To(Equal("0123456789abcdefghijklmnoqrstu"))

			// buffer has unwritten data so close should fail
			err = writeAtBuffer.Close()
			Expect(err).To(HaveOccurred())

			writeAtBuffer.WriteAt([]byte("vw"), 30)
			Expect(output.String()).To(Equal("0123456789abcdefghijklmnoqrstuvwxyz"))

			err = writeAtBuffer.Close()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
