// Copyright GoSed (c) 2021, Carter Peel 
// This code is licensed under MIT license (see LICENSE for details)

package gosed

import (
	"bufio"
	"io"

	//"bytes"
	"fmt"
	"github.com/jf-tech/go-corelib/ios"
	"github.com/zenthangplus/goccm"
	"log"
	"os"
	"sync"
	"time"
)

// Replacer contains all of the methods needed to properly execute replace operations
type Replacer struct {
	Config *replacerConfig
}

// replacerConfig contains all of the config variables
type replacerConfig struct {
	File         *os.File
	FilePath     string
	FileSize     int64
	Asynchronous bool
	Mappings     *replacerMappings
	Semaphore    *replacerSemaphore
}

// replacerStringMappings maps old byte sequences to new byte sequences
type replacerMappings struct {
	Keys    [][]byte
	Indices [][]byte
}

// replacerSemaphore contains all of the channels and waitgroups needed for async
type replacerSemaphore struct {
	Waiter    sync.WaitGroup
	Operating chan struct{}
	Done      chan bool
	GCM       goccm.ConcurrencyManager
	JobWaiter chan bool
	Used      bool
	Queue     int
}

// NewReplacer returns a new *Replacer struct
func NewReplacer(fileName string) (*Replacer, error) {
	fi, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	fiStat, err := fi.Stat()
	if err != nil {
		return nil, err
	}
	return &Replacer{
		Config: &replacerConfig{
			File:     fi,
			FilePath: fileName,
			FileSize: fiStat.Size(),
			Mappings: &replacerMappings{
				Keys:    make([][]byte, 0),
				Indices: make([][]byte, 0),
			},
			Asynchronous: false,
			Semaphore: &replacerSemaphore{
				GCM:    goccm.New(1),
				Waiter: sync.WaitGroup{},
			},
		},
	}, nil
}

// NewMapping maps a new oldString:newString []byte entry
func (rp *Replacer) NewMapping(oldString, newString []byte) error {
	switch {
	case len(oldString) == 0:
		return fmt.Errorf("cannot replace empty string with new value")
	}
	rp.Config.Mappings.Keys = append(rp.Config.Mappings.Keys, oldString)
	rp.Config.Mappings.Indices = append(rp.Config.Mappings.Indices, newString)
	return nil
}

// NewStringMapping maps a new oldString:newString string entry
func (rp *Replacer) NewStringMapping(oldString, newString string) error {
	switch {
	case oldString == "":
		return fmt.Errorf("cannot replace empty string with new value")
	}
	rp.Config.Mappings.Keys = append(rp.Config.Mappings.Keys, []byte(oldString))
	rp.Config.Mappings.Indices = append(rp.Config.Mappings.Indices, []byte(newString))
	return nil
}

// Replace does the replace operation on the file
func (rp *Replacer) Replace() (int, error) {
	var count int
	n, err := DoReplace(rp)
	if err != nil {
		return n, err
	}
	count += n
	return count, nil
}

func (rp *Replacer) Reset() error {
	var err error
	if err := rp.Config.File.Close(); err != nil {
		return err
	}
	rp.Config.File, err = os.OpenFile(rp.Config.FilePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	rp.Config.Mappings.Keys = rp.Config.Mappings.Keys[:0]
	rp.Config.Mappings.Indices = rp.Config.Mappings.Indices[:0]
	return nil
}

// DoReplace does the replace operation
func DoReplace(rp *Replacer) (int, error) {
	tmpfile := fmt.Sprintf("tmp-gosed-%d", time.Now().UnixNano())
	input, err := os.OpenFile(rp.Config.FilePath, os.O_RDWR, 0644)
	if err != nil {
		log.Printf("Error opening file: %s\n", err.Error())
		return 0, err
	}
	output, err := os.OpenFile(tmpfile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Error opening file: %s\n", err.Error())
		return 0, err
	}
	defer func(input, output *os.File, tmpFile string) {
		if err := input.Close(); err != nil {
			log.Printf("Error closing input: %s\n", err.Error())
		}
		if err := output.Close(); err != nil {
			log.Printf("Error closing output: %s\n", err.Error())
		}
	}(input, output, tmpfile)
	var replacer = ios.NewBytesReplacingReader(bufio.NewReader(input), rp.Config.Mappings.Keys[0], rp.Config.Mappings.Indices[0])
	for index, key := range rp.Config.Mappings.Keys {
		if index == 0 {
			continue
		}
		replacer = ios.NewBytesReplacingReader(replacer, key, rp.Config.Mappings.Indices[index])
	}
	wrote, err := io.CopyBuffer(output, replacer, make([]byte, 8192))
	if err != nil {
		log.Printf("Error copying: %s\n", err.Error())
		return 0, err
	}
	if err := os.Remove(rp.Config.FilePath); err != nil {
		return 0, err
	}
	if err := os.Rename(tmpfile, rp.Config.FilePath); err != nil {
		return 0, err
	}
	return int(wrote), nil
}
