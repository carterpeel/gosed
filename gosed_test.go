package gosed

import (
	"crypto/sha256"
	"fmt"
	"github.com/tjarratt/babble"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSmall(t *testing.T) {
	defer Cleanup()
	babbler := babble.NewBabbler()
	var wordlist = make([]string, 0)
	for i := 0; i < 24; i++ {
		wordlist = append(wordlist, babbler.Babble())
	}

	fi, err := os.OpenFile("test.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 50000; i++ {
		_, err := fi.WriteString(wordlist[rand.Intn(23)])
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	if err = fi.Close(); err != nil {
		t.Fatal(err.Error())
	}

	replacer, err := NewReplacer("test.txt")
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := replacer.NewStringMapping(wordlist[0], "REPLACED"); err != nil {
		t.Fatal(err.Error())
	}
	start := time.Now()
	replaced, err := replacer.Replace()
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Printf("replaced %d bytes in %s\n", replaced, time.Since(start))
}

func TestFull(t *testing.T) {
	defer Cleanup()
	babbler := babble.NewBabbler()
	var wordlist []string
	fi, err := os.OpenFile("template.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
	babbler.Count = 10
	babbler.Separator = "-"
	for i := 0; i < 1024; i++ {
		wordlist = append(wordlist, babbler.Babble())
	}
	rand.Seed(time.Now().UnixNano())
	if len(wordlist) == 0 {
		t.Fatal(fmt.Errorf("wordlist cannot have a len() of 0"))
	}
	for i := 0; i < 250000; i++ {
		_, err := fi.WriteString(wordlist[rand.Intn(len(wordlist))])
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	err = fi.Sync()
	if err != nil {
		t.Fatal(err.Error())
	}
	if err = fi.Close(); err != nil {
		t.Fatal(err.Error())
	}
	fi, err = os.Open("template.txt")
	if err != nil {
		t.Fatal(err.Error())
	}
	fi2, err := os.OpenFile("test-sed.txt", os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
	fi3, err := os.OpenFile("test-gosed.txt", os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
	if _, err = io.Copy(io.MultiWriter(fi2, fi3), fi); err != nil {
		t.Fatal(err.Error())
	}
	replacer, err := NewReplacer("test-gosed.txt")
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := fi2.Close(); err != nil {
		t.Fatal(err.Error())
	}
	if err := fi3.Close(); err != nil {
		t.Fatal(err.Error())
	}
	for i, v := range wordlist {
		if err := replacer.NewStringMapping(v, fmt.Sprintf("REPLACED-%d", i)); err != nil {
			t.Fatal(err.Error())
		}
	}
	start := time.Now()
	replaced, err := replacer.Replace()
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Printf("[gosed] --> replaced %d occurrences in %s\n", replaced, time.Since(start))
	var args = fmt.Sprintf("#!/bin/bash\n/usr/local/bin/gsed -i '")
	for i, v := range wordlist {
		if i != len(wordlist)-1 {
			args = fmt.Sprintf("%ss/%s/REPLACED-%d/g; ", args, v, i)
		} else if i == len(wordlist)-1 {
			args = fmt.Sprintf("%ss/%s/REPLACED-%d/g;' test-sed.txt", args, v, i)
		}
	}
	if err := ioutil.WriteFile("./gsed.sh", []byte(args), 0777); err != nil {
		t.Fatal(err.Error())
	}
	start = time.Now()
	cmd := exec.Command("./gsed.sh")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		log.Println(strings.Join(cmd.Args, " "))
		t.Fatal(err.Error())
	}
	log.Printf("[gnused] --> replaced all occurrences in %s\n", time.Since(start))
	log.Println("Comparing files...")
	hasher1 := sha256.New()
	hasher2 := sha256.New()
	hashReader1, err := os.Open("test-gosed.txt")
	if err != nil {
		log.Printf("Error opening gosed test file: %s\n", err.Error())
		t.Fatal(err.Error())
	}
	hashReader2, err := os.Open("test-sed.txt")
	if err != nil {
		log.Printf("Error opening gnused test file: %s\n", err.Error())
		t.Fatal(err.Error())
	}
	if _, err = io.Copy(hasher1, hashReader1); err != nil {
		t.Fatal(err.Error())
	}
	if _, err = io.Copy(hasher2, hashReader2); err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("%x:test-gosed.txt\n%x:test-sed.txt\n", hasher1.Sum(nil), hasher2.Sum(nil))
	if string(hasher1.Sum(nil)) != string(hasher2.Sum(nil)) {
		fmt.Printf("%x:test-gosed.txt\n%x:test-sed.txt", hasher1.Sum(nil), hasher2.Sum(nil))
		t.Fatal(fmt.Errorf("file hashes do not match"))
	}
}

func Cleanup() {
	files, err := filepath.Glob("*.txt")
	if err != nil {
		panic(err.Error())
	}
	for _, fi := range files {
		_ = os.Remove(fi)
	}
	files, err = filepath.Glob("*.sh")
	if err != nil {
		panic(err.Error())
	}
	for _, fi := range files {
		_ = os.Remove(fi)
	}
}
