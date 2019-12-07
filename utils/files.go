package utils

import (
	"fmt"
	"io"
)

// Searches from the given path for a directory of the given name.
// If the directory is found in the user's home directory then it is
// returned. If not found the the search continues up the directory
// tree. If directory is not found then it will be created at the
// given path.
func DirReverseLookup(name, homePath string) string {

	return ""
}

// Chunk seeker
type chunkSeeker struct {
	offset,
	chunkSize,
	ptr,
	remainder int64
}

func (s *chunkSeeker) Seek(offset int64, whence int) (int64, error) {

	switch whence {
	case io.SeekStart:
		s.ptr = offset
	case io.SeekCurrent:
		s.ptr = s.ptr + offset
	case io.SeekEnd:
		s.ptr = s.chunkSize + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if s.ptr < 0 {
		return 0, fmt.Errorf("negative position")
	}

	s.remainder = s.chunkSize - s.ptr
	return s.ptr, nil
}

// Chunk reader seeker
type ChunkReadSeeker struct {
	chunkSeeker

	source io.ReaderAt
}

func NewChunkReadSeeker(source io.ReaderAt, offset, chunkSize int64) io.ReadSeeker {

	return &ChunkReadSeeker{
		chunkSeeker: chunkSeeker{
			offset:    offset,
			chunkSize: chunkSize,

			ptr:       0,         // current position in chunk
			remainder: chunkSize, // unprocessed space in chunk
		},

		source: source,
	}
}

func (r *ChunkReadSeeker) Read(data []byte) (int, error) {

	var (
		err  error
		size int64
		n    int
	)

	size = int64(len(data))
	if r.remainder > size {
		n, err = r.source.ReadAt(data, r.offset+r.ptr)

		size = int64(n)
		if err != io.EOF {
			r.remainder = r.remainder - size
		} else {
			r.remainder = 0
		}
		r.ptr = r.ptr + size
		return n, err

	} else if r.remainder > 0 {
		if n, err = r.source.ReadAt(data[0:r.remainder], r.offset+r.ptr); err == nil {
			// no more bytes in chunk so return EOF
			err = io.EOF
		}

		r.remainder = 0
		r.ptr = r.ptr + int64(n)
		return n, err
	}
	return 0, io.EOF
}

// Chunk write
type ChunkWriteSeeker struct {
	chunkSeeker

	dest io.WriterAt
}

func NewChunkWriteSeeker(dest io.WriterAt, offset, chunkSize int64) io.WriteSeeker {

	return &ChunkWriteSeeker{
		chunkSeeker: chunkSeeker{
			offset:    offset,
			chunkSize: chunkSize,

			ptr:       0,         // current position in chunk
			remainder: chunkSize, // unprocessed space in chunk
		},

		dest: dest,
	}
}

func (r *ChunkWriteSeeker) Write(data []byte) (int, error) {

	var (
		err  error
		size int64
		n    int
	)

	size = int64(len(data))
	if r.remainder > size {
		n, err = r.dest.WriteAt(data, r.offset+r.ptr)

		size = int64(n)
		r.remainder = r.remainder - size
		r.ptr = r.ptr + size
		return n, err

	} else if r.remainder > 0 {
		n, err = r.dest.WriteAt(data[0:r.remainder], r.offset+r.ptr)

		r.remainder = 0
		r.ptr = r.ptr + int64(n)
		return n, err
	}
	return 0, nil
}

func (r *ChunkWriteSeeker) Seek(offset int64, whence int) (int64, error) {

	switch whence {
	case io.SeekStart:
		r.ptr = offset
	case io.SeekCurrent:
		r.ptr = r.ptr + offset
	case io.SeekEnd:
		r.ptr = r.chunkSize + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}
	if r.ptr < 0 {
		return 0, fmt.Errorf("negative position")
	}

	r.remainder = r.chunkSize - r.ptr
	return r.ptr, nil
}
