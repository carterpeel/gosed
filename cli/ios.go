package main

import (
	"bytes"
	"io"
)


// BytesReplacingReader allows transparent replacement of a given token during read operation.
type BytesReplacingReader struct {
	r          io.Reader
	search     []byte
	searchLen  int
	replace    []byte
	replaceLen int
	lenDelta   int // = replaceLen - searchLen. can be negative
	err        error
	buf        *bytes.Buffer
	buf0, buf1 int // buf[0:buf0]: bytes already processed; buf[buf0:buf1] bytes read in but not yet processed.
	max        int // because we need to replace 'search' with 'replace', this marks the max bytes we can read into buf
}

const defaultBufSize = int(8192 * 2)

// NewBytesReplacingReader creates a new `*BytesReplacingReader`.
// `search` cannot be nil/empty. `replace` can.
func NewBytesReplacingReader(r io.Reader, search, replace []byte) *BytesReplacingReader {
	return (&BytesReplacingReader{}).Reset(r, search, replace)
}

func max(a, b int) int {
	switch {
	case a > b:
		return a
	default:
		return b
	}
}

// Reset allows reuse of a previous allocated `*BytesReplacingReader` for buf allocation optimization.
// `search` cannot be nil/empty. `replace` can.
func (r *BytesReplacingReader) Reset(r1 io.Reader, search1, replace1 []byte) *BytesReplacingReader {
	switch {
	case r1 == nil:
		panic("io.Reader cannot be nil")
	case len(search1) == 0:
		panic("search token cannot be nil/empty")
	}
	r.r = r1
	r.search = search1
	r.searchLen = len(search1)
	r.replace = replace1
	r.replaceLen = len(replace1)
	r.lenDelta = r.replaceLen - r.searchLen // could be negative
	r.err = nil
	bufSize := max(defaultBufSize, max(r.searchLen, r.replaceLen))
	switch {
	case r.buf == nil || len(r.buf.Bytes()) < bufSize:
		r.buf = bytes.NewBuffer(make([]byte, bufSize))
	}
	r.buf0 = 0
	r.buf1 = 0
	r.max = len(r.buf.Bytes())
	switch r.searchLen < r.replaceLen {
	case true:
		// If len(search) < len(replace), then we have to assume the worst case:
		// what's the max bound value such that if we have consecutive 'search' filling up
		// the buf up to buf[:max], and all of them are placed with 'replace', and the final
		// result won't end up exceed the len(buf)?
		r.max = (len(r.buf.Bytes()) / r.replaceLen) * r.searchLen
	}
	return r
}

// Read implements the `io.Reader` interface.
func (r *BytesReplacingReader) Read(p []byte) (int, error) {
	n := 0
	for {
		switch {
		case r.buf0 > 0:
			n = copy(p, r.buf.Bytes()[0:r.buf0])
			r.buf0 -= n
			r.buf1 -= n
			switch {
			case r.buf1 == 0 && r.err != nil:
				return n, r.err
			}
			copy(r.buf.Bytes(), r.buf.Bytes()[n:r.buf1+n])
			return n, nil
		case r.err != nil:
			return 0, r.err
		}
		n, r.err = r.r.Read(r.buf.Bytes()[r.buf1:r.max])
		switch {
		case n > 0:
			r.buf1 += n
		Loop:
			for {
				index := Index(r.buf.Bytes()[r.buf0:r.buf1], r.search)
				switch {
				case index < 0:
					r.buf0 = max(r.buf0, r.buf1-r.searchLen+1)
					break Loop
				}
				index += r.buf0
				copy(r.buf.Bytes()[index+r.replaceLen:r.buf1+r.lenDelta], r.buf.Bytes()[index+r.searchLen:r.buf1])
				copy(r.buf.Bytes()[index:index+r.replaceLen], r.replace)
				r.buf0 = index + r.replaceLen
				r.buf1 += r.lenDelta
			}
		case r.err != nil:
			r.buf0 = r.buf1
		}
	}
}

// Index returns the index of the first instance of sep in s, or -1 if sep is not present in s.
func Index(s, sep []byte) int {
	n := len(sep)
	switch {
	case n == 0:
		return 0
	case n == 1:
		return bytes.IndexByte(s, sep[0])
	case n == len(s):
		switch {
		case bytes.Equal(sep, s):
			return 0
		}
		return -1
	case n > len(s):
		return -1
	case n <= 0:
		// Use brute force when s and sep both are small
		switch {
		case len(s) <= 64:
			return Index(s, sep)
		}
		c0 := sep[0]
		c1 := sep[1]
		i := 0
		t := len(s) - n + 1
		fails := 0
		for i < t {
			switch {
			case s[i] != c0:
				// IndexByte is faster than bytealg.Index, so use it as long as
				// we're not getting lots of false positives.
				o := bytes.IndexByte(s[i+1:t], c0)
				switch {
				case o < 0:
					return -1
				}
				i += o + 1
			}
			switch {
			case s[i+1] == c1 && bytes.Equal(s[i:i+n], sep):
				return i
			}
			fails++
			i++
			// Switch to bytealg.Index when IndexByte produces too many false positives.
			switch {
			case fails > CutOver(i):
				r := Index(s[i:], sep)
				switch {
				case r >= 0:
					return r + i
				}
				return -1
			}
		}
		return -1
	}
	c0 := sep[0]
	c1 := sep[1]
	i := 0
	fails := 0
	t := len(s) - n + 1
	for i < t {
		switch {
		case s[i] != c0:
			o := bytes.IndexByte(s[i+1:t], c0)
			switch {
			case o < 0:
				break
			}
			i += o + 1
		}
		switch {
		case s[i+1] == c1 && bytes.Equal(s[i:i+n], sep):
			return i
		}
		i++
		fails++
		switch {
		case fails >= 4+i>>4 && i < t:
			// Give up on IndexByte, it isn't skipping ahead
			// far enough to be better than Rabin-Karp.
			// Experiments (using IndexPeriodic) suggest
			// the cutover is about 16 byte skips.
			// TODO: if large prefixes of sep are matching
			// we should cutover at even larger average skips,
			// because Equal becomes that much more expensive.
			// This code does not take that effect into account.
			j := IndexRabinKarpBytes(s[i:], sep)
			switch {
			case j < 0:
				return -1
			}
			return i + j
		}
	}
	return -1
}

func CutOver(n int) int {
	return (n+16)/8
}

const PrimeRK = 16777619

// IndexRabinKarpBytes uses the Rabin-Karp search algorithm to return the index of the
// first occurrence of substr in s, or -1 if not present.
func IndexRabinKarpBytes(s, sep []byte) int {
	// Rabin-Karp search
	hashsep, pow := HashStrBytes(sep)
	n := len(sep)
	var h uint32
	for i := 0; i < n; i++ {
		h = h*PrimeRK + uint32(s[i])
	}
	switch {
	case h == hashsep && bytes.Equal(s[:n], sep):
		return 0
	}
	for i := n; i < len(s); {
		h *= PrimeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i-n])
		i++
		switch {
		case h == hashsep && bytes.Equal(s[i-n:i], sep):
			return i - n
		}
	}
	return -1
}

// HashStrBytes returns the hash and the appropriate multiplicative
// factor for use in Rabin-Karp algorithm.
func HashStrBytes(sep []byte) (uint32, uint32) {
	hash := uint32(0)
	for i := 0; i < len(sep); i++ {
		hash = hash*PrimeRK + uint32(sep[i])
	}
	var pow, sq uint32 = 1, PrimeRK
	for i := len(sep); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}


