package gosed

import (
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"log"
)

func ReplaceIn(f *os.File, original, new []byte) error {
	switch {
	case len(original) == 0:
		return fmt.Errorf("empty search parameter disallowed")
	case f == nil:
		return fmt.Errorf("cannot use nil value as argument for file")
	}

	fStat, err := f.Stat()
	if err != nil {
		return err
	}
	switch {
	case fStat.IsDir():
		return fmt.Errorf("cannot replace strings in a directory")
	case f.Fd() == unix.O_RDONLY:
		return fmt.Errorf("cannot replace strings in read only file")
	case f.Fd() == unix.O_TRUNC:
		return fmt.Errorf("cannot safely perform operation with os.O_TRUNC descriptor")
	case f.Fd() == unix.O_RDWR:
		break
	}


	in := make([]byte, 1)
	readBuffer := make([]byte, 0) // when new > original, we have to read the bytes we will override
	matchIndex := 0
	newLen := 0
	readDone := false
	// If file is not at eof, read from file. else, read from our buffer if it's not empty
	DoRead := func(b []byte, index int64, appendReadBuffer bool) (int, error) {
		if readDone {
			if appendReadBuffer {
				return 0, nil
			}
			if len(readBuffer) == 0 {
				return 0, io.EOF
			}
			n := 0
			for ; n < len(b) && n < len(readBuffer); n++ {
				b[n] = readBuffer[n]
			}
			readBuffer = readBuffer[n:]
			//fmt.Println("DoRead:", len(b), n, string(b[:n]), string(readBuffer))
			return n, nil
		} else {
			n, err := f.ReadAt(b, index)
			if err != nil {
				readDone = true
			}
			if appendReadBuffer {
				return n, nil
			}
			readBuffer = append(readBuffer, b[:n]...)
			if n < len(b) {
				if len(b) > len(readBuffer) {
					n = len(readBuffer)
				} else {
					n = len(b)
				}
			}
			for i, v := range readBuffer[:n] {
				b[i] = v
			}
			readBuffer = readBuffer[n:]
			//fmt.Println("DoRead:", len(b), n, string(b[:n]), string(readBuffer))
			if n != 0 {
				err = nil
			}
			return n, err
		}
	}
	var ri, wi int64
	for {
		_, err := DoRead(in, ri, false)
		if err != nil {
			break
		}
		//fmt.Printf("in:%v ri:%v wi:%v matchIndex:%v readBuffer:%v newLen:%v\n", string(in), ri, wi, matchIndex, string(readBuffer), newLen)
		ri++
		if in[0] == original[matchIndex] {
			matchIndex++
			if matchIndex == len(original) {
				if len(new) > len(original) {
					in := make([]byte, len(new)-len(original))
					n, _ := DoRead(in, ri, true) // only to manage buffer
					ri += int64(n)
					readBuffer = append(readBuffer, in[:n]...)
				}
				_, _ = f.WriteAt(new, wi)
				matchIndex = 0
				newLen += len(new)
				wi += int64(len(new))
				//fmt.Println("matched")
			}
		} else if matchIndex != 0 {
			//fmt.Println("partial match reset")
			n, _ := f.WriteAt(original[:matchIndex], wi)
			wi += int64(n)
			newLen += n
			if in[0] == original[0] {
				matchIndex = 1
			} else {
				_, err = f.WriteAt(in, wi)
				if err != nil {
					log.Printf("Error writing '%s' at offset '%d': %s\n", string(in), wi, err.Error())
					// We don't return/continue here because the parent conditional will already continue to the next iteration after updating the high-scope variables
				}
				wi++
				newLen++
				matchIndex = 0
			}
		} else {
			_, err = f.WriteAt(in, wi)
			if err != nil {
				log.Printf("Error writing '%s' at offset '%d': %s\n", string(in), wi, err.Error())
				// We don't return/continue here because the parent conditional will already continue to the next iteration after updating the high-scope variables
			}
			wi++
			newLen++
		}
	}
	err = f.Truncate(int64(newLen))
	if err != nil {
		log.Printf("Error truncating file: %s\n", err.Error())
		return err
	}
	return nil
}
