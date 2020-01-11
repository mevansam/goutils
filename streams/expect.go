package streams

import (
	"bytes"
	"io"
	"regexp"
)

type ExpectStream struct {
	inputStream  io.Reader // input stream for data from client
	outputStream io.Writer // outpt stream for data to client

	streamReader io.Reader // stream on which to match expact patterns
	streamWriter io.Writer // stream to which expect commands will be written

	serialExpects []*expect
	inSerial      bool

	// triggers on output stream
	triggerOutExpects []*expect
	activeOutTriggers []*expect

	// triggers on input stream
	triggerInExpects []*expect
	activeInTriggers []*expect

	// internal buffer size (default 512)
	bufferSize int

	// internal error
	err error
}

type expect struct {
	startPattern,
	endPattern *regexp.Regexp

	command string
}

// the expect stream will search for expect patterns
// in the input stream and respond with commands in
// the output stream with a response associated
// with that pattern. patterns and their commands
// are evaluated in order.
func NewExpectStream(
	inputStream io.Reader, // i.e. standard in of client
	outputStream io.Writer, // i.e. standard out of client
) (
	*ExpectStream,
	io.ReadCloser, // i.e. replaces standard in for data to receiver
	io.WriteCloser, // i.e. replaces standard out for data from receiver
) {

	var (
		pipedReadStream  io.ReadCloser  // piped input for data to receiver
		pipedWriteStream io.WriteCloser // piped output for data from receiver
	)

	es := &ExpectStream{
		inputStream:  inputStream,
		outputStream: outputStream,

		serialExpects: []*expect{},
		inSerial:      false,

		triggerOutExpects: []*expect{},
		activeOutTriggers: []*expect{},

		triggerInExpects: []*expect{},
		activeInTriggers: []*expect{},

		bufferSize: 512,
	}

	pipedReadStream, es.streamWriter = io.Pipe()
	es.streamReader, pipedWriteStream = io.Pipe()

	return es, pipedReadStream, pipedWriteStream
}

func (es *ExpectStream) SetBufferSize(size int) {
	es.bufferSize = size
}

func (es *ExpectStream) AddExpect(
	pattern, command string,
	serial bool,
) {
	if serial {
		es.serialExpects = append(
			es.serialExpects,
			&expect{
				startPattern: regexp.MustCompile(pattern),
				endPattern:   nil,
				command:      command,
			},
		)
	} else {
		es.triggerOutExpects = append(
			es.triggerOutExpects,
			&expect{
				startPattern: regexp.MustCompile(pattern),
				endPattern:   nil,
				command:      command,
			},
		)
	}
}

func (es *ExpectStream) AddMultiLineExpect(
	startPattern, endPattern, command string,
	serial bool,
) {
	if serial {
		es.serialExpects = append(
			es.serialExpects,
			&expect{
				startPattern: regexp.MustCompile(startPattern),
				endPattern:   regexp.MustCompile(endPattern),
				command:      command,
			},
		)
	} else {
		es.triggerOutExpects = append(
			es.triggerOutExpects,
			&expect{
				startPattern: regexp.MustCompile(startPattern),
				endPattern:   regexp.MustCompile(endPattern),
				command:      command,
			},
		)
	}
}

func (es *ExpectStream) AddExpectInTrigger(
	pattern, command string,
) {
	es.triggerInExpects = append(
		es.triggerInExpects,
		&expect{
			startPattern: regexp.MustCompile(pattern),
			endPattern:   nil,
			command:      command,
		},
	)
}

func (es *ExpectStream) AddMultiLineExpectInTrigger(
	startPattern, endPattern, command string,
) {
	es.triggerOutExpects = append(
		es.triggerOutExpects,
		&expect{
			startPattern: regexp.MustCompile(startPattern),
			endPattern:   regexp.MustCompile(endPattern),
			command:      command,
		},
	)
}

func (es *ExpectStream) Start() {

	go func() {

		var (
			err error

			i, j, l int
			newLine bool

			lineBuffer bytes.Buffer
			line       []byte
		)

		lineBuffer.Grow(es.bufferSize)
		buffer := make([]byte, es.bufferSize)

		for err == nil {
			// read until a newline is encountered in bytes read
			if l, err = es.streamReader.Read(buffer); err != nil && err != io.EOF {
				break
			}
			// echo data from reciever to client's output
			if _, err = es.outputStream.Write(buffer[0:l]); err != nil {
				break
			}

			for i = 0; i < l; {
				newLine = false
				for j = i; j < l; j++ {
					if buffer[j] == '\n' {
						newLine = true
						break
					}
				}

				lineBuffer.Write(buffer[i:j])
				line = lineBuffer.Bytes()

				if err = es.processSerialExpects(line); err != nil {
					break
				}

				if newLine {
					// if new line then we reset the line
					// buffer and start building a new line
					lineBuffer.Reset()
					i = j + 1
				} else {
					i = j
				}
			}
		}
		if err != io.EOF {
			es.err = err
		}
	}()
}

func (es *ExpectStream) processSerialExpects(line []byte) error {

	var (
		err error

		curExpect *expect
	)

	if len(es.serialExpects) > 0 {
		curExpect = es.serialExpects[0]

		if !es.inSerial {
			// match start pattern of expect
			if curExpect.startPattern.Match(line) {
				es.inSerial = true
			}
		}
		if es.inSerial {
			// match end pattern of expect if one exists
			if curExpect.endPattern == nil ||
				curExpect.endPattern.Match(line) {

				// send command to reciever
				if _, err = es.streamWriter.Write([]byte(curExpect.command)); err != nil {
					return err
				}

				// pop the expect at the top of the stack
				es.serialExpects = es.serialExpects[1:]
				es.inSerial = false
			}
		}
	}

	return nil
}
