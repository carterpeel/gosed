package gosed

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

// Tests to see if the core functionality works
func TestReplaceIn(t *testing.T) {
	//defer t.Cleanup(Cleanup)
	fi, err := os.OpenFile("bigAssFile.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		t.Errorf("Error opening file to generate test data: %s\n", err.Error())
		return
	}
	for i := 0; i <= 500; i++ {
		if _, err = fi.WriteString("OLDSTRING-"); err != nil {
			t.Errorf("Error writing test data: %s\n", err.Error())
			return
		}
	}
	if err := fi.Close(); err != nil {
		t.Errorf("Error closing test file: %s\n", err.Error())
		return
	}
	fi, err = os.OpenFile("bigAssFile.txt", os.O_RDWR, 0777)
	if err != nil {
		t.Errorf("Error opening file to test ReplaceIn function: %s\n", err.Error())
		return
	}
	if err := ReplaceIn(fi, []byte("OLDSTRING-"), []byte("NEWSTRING-")); err != nil {
		t.Errorf("Error replacing in file: %s\n", err.Error())
		return
	}
}

// Tests to see if ReplaceIn() will handle a nil file pointer properly
func TestReplaceInWithNilFile(t *testing.T) {
	if err := ReplaceIn(nil, []byte("OLDSTRING-"), []byte("NEWSTRING-")); err == nil {
		t.Error("'gosed.ReplaceIn()' did not return the proper error upon invocation with a nil *os.File pointer")
		return
	}
}

func TestReplaceInConcurrently(t *testing.T) {
	fi, err := os.OpenFile("bigAssFile.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		t.Errorf("Error opening file to generate test data: %s\n", err.Error())
		return
	}
	if err = fi.Truncate(0); err != nil {
		t.Errorf("Error truncating file: %s\n", err.Error())
		return
	}
	for i := 0; i <= 10; i++ {
		if _, err = fi.WriteString("OLDSTRING0-"); err != nil {
			t.Errorf("Error writing test data: %s\n", err.Error())
			return
		}
	}
	if err := fi.Close(); err != nil {
		t.Errorf("Error closing test file: %s\n", err.Error())
		return
	}
	fi, err = os.OpenFile("bigAssFile.txt", os.O_RDWR, 0777)
	if err != nil {
		t.Errorf("Error opening file to test ReplaceIn function: %s\n", err.Error())
		return
	}
	var switcher int
	var waitg = &sync.WaitGroup{}
	for i := 0; i < 64; i++ {
		waitg.Add(1)
		time.Sleep(10 * time.Millisecond)
		go func(waitg *sync.WaitGroup, switcher int) {
			defer waitg.Done()
			log.Printf("Replacing OLDSTRING%d- with OLDSTRING%d-", switcher-1, switcher)
			if err := ReplaceIn(fi, []byte(fmt.Sprintf("OLDSTRING%d-", switcher-1)), []byte(fmt.Sprintf("OLDSTRING%d-", switcher))); err != nil {
				t.Errorf("Error replacing in file concurrently: %s\n", err.Error())
				return
			}
		}(waitg, switcher)
		switcher++
	}
	waitg.Wait()
	fb, err := ioutil.ReadFile("bigAssFile.txt")
	if err != nil {
		t.Errorf("Error reading file: %s\n", err.Error())
		return
	}
	log.Println(string(fb))
	t.Cleanup(Cleanup)
}

// Cleans up the working dir
func Cleanup() {
	if err := os.Remove("bigAssFile.txt"); err != nil {
		log.Fatal(err.Error())
	}
}
