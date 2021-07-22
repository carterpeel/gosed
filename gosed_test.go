package gosed

import (
	"log"
	"os"
	"testing"
)

// Tests to see if the core functionality works
func TestReplaceIn(t *testing.T) {
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
	defer t.Cleanup(Cleanup)
	if err := ReplaceIn(nil, []byte("OLDSTRING-"), []byte("NEWSTRING-")); err == nil {
		t.Error("'gosed.ReplaceIn()' did not return the proper error upon invocation with a nil *os.File pointer")
		return
	}
}


// Cleans up the working dir
func Cleanup() {
	if err := os.Remove("bigAssFile.txt"); err != nil {
		log.Fatal(err.Error())
	}
}



