package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
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
		switch {
		case readDone:
			switch {
			case appendReadBuffer:
				return 0, nil
			case len(readBuffer) == 0:
				return 0, io.EOF
			}
			n := 0
			for ; n < len(b) && n < len(readBuffer); n++ {
				b[n] = readBuffer[n]
			}
			readBuffer = readBuffer[n:]
			//fmt.Println("DoRead:", len(b), n, string(b[:n]), string(readBuffer))
			return n, nil
		default:
			n, err := f.ReadAt(b, index)
			switch {
			case err != nil:
				readDone = true
			case appendReadBuffer:
				return n, nil
			}
			readBuffer = append(readBuffer, b[:n]...)
			switch {
			case n < len(b):
				switch {
				case len(b) > len(readBuffer):
					n = len(readBuffer)
				default:
					n = len(b)
				}
			}
			for i, v := range readBuffer[:n] {
				b[i] = v
			}
			readBuffer = readBuffer[n:]
			//fmt.Println("DoRead:", len(b), n, string(b[:n]), string(readBuffer))
			switch {
			case n != 0:
				return n, nil
			default:
				return n, err
			}
		}
	}
	var ri, wi int64
	for {
		if _, err := DoRead(in, ri, false); err != nil {
			break
		}
		//fmt.Printf("in:%v ri:%v wi:%v matchIndex:%v readBuffer:%v newLen:%v\n", string(in), ri, wi, matchIndex, string(readBuffer), newLen)
		ri++
		switch {
		case in[0] == original[matchIndex]:
			matchIndex++
			switch {
			case matchIndex == len(original):
				switch {
				case len(new) > len(original):
					in := make([]byte, len(new)-len(original))
					n, _ := DoRead(in, ri, true) // only to manage buffer
					ri += int64(n)
					readBuffer = append(readBuffer, in[:n]...)
				}
				_, _ = f.WriteAt(new, wi)
				matchIndex = 0
				newLen += len(new)
				wi += int64(len(new))
			}
		case matchIndex != 0:
			//fmt.Println("partial match reset")
			n, _ := f.WriteAt(original[:matchIndex], wi)
			wi += int64(n)
			newLen += n
			switch {
			case in[0] == original[0]:
				matchIndex = 1
			default:
				if _, err = f.WriteAt(in, wi); err != nil {
					// We don't return/continue here because the parent conditional will already continue to the next iteration after updating the high-scope variables
					// Do nothing because this is a module and we don't need to return this error
					//log.Printf("Error writing '%s' at offset '%d': %s\n", string(in), wi, err.Error())
				}
				wi++
				newLen++
				matchIndex = 0
			}
		default:
			if _, err = f.WriteAt(in, wi); err != nil {
				// Do nothing because this is a module and we don't need to return this error
				//log.Printf("Error writing '%s' at offset '%d': %s\n", string(in), wi, err.Error())
			}
			wi++
			newLen++
		}
	}
	if err = f.Truncate(int64(newLen)); err != nil {
		//log.Printf("Error truncating file: %s\n", err.Error())
		return err
	}
	return nil
}
