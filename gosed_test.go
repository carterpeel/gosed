package gosed

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/carterpeel/go-corelib/ios"
	"github.com/docker/go-units"
	"github.com/tjarratt/babble"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTiny(t *testing.T) {
	start := time.Now()
	tinyBytes := []byte{'a', 'b', 'c', 'a', 'd'}
	newBytes, err := ioutil.ReadAll(ios.NewBytesReplacingReader(bytes.NewReader(tinyBytes), []byte("a"), []byte("f")))
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("Old Bytes: %s\nNew Bytes: %s\nTook %s\n", string(tinyBytes), string(newBytes), time.Since(start))
	if !bytes.Equal(newBytes, []byte{'f', 'b', 'c', 'f', 'd'}) {
		fmt.Printf("Old Bytes: %s\nNew Bytes: %s\n", string(tinyBytes), string(newBytes))
		t.Fatal(fmt.Errorf("new bytes did not match"))
	}
}


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
	if err := copyFileContents("test.txt", "test-gosed.txt"); err != nil {
		t.Fatal(err.Error())
	}
	if err := copyFileContents("test.txt", "test-sed.txt"); err != nil {
		t.Fatal(err.Error())
	}
	replacer, err := NewReplacer("test-gosed.txt")
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := replacer.NewStringMapping(wordlist[0], "REPLACED"); err != nil {
		t.Fatal(err.Error())
	}
	start := time.Now()
	replaced, err := replacer.ReplaceChained()
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("[gosed] --> replaced %d bytes in %s\n", replaced, time.Since(start))
	var sedPath string
	if runtime.GOOS == "darwin" {
		sedPath = "gsed"
	} else {
		sedPath = "sed"
	}
	start = time.Now()
	out, err := exec.Command(sedPath, "-i", fmt.Sprintf("s/%s/REPLACED/g", wordlist[0]), "test-sed.txt").CombinedOutput()
	if err != nil {
		log.Printf("gnused output: %s\n", string(out))
		t.Fatal(err.Error())
	}
	fmt.Printf("[gnused] --> replaced all occurences in %s\n", time.Since(start))
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

func TestSequentialSmall(t *testing.T) {
	defer Cleanup()
	babbler := babble.NewBabbler()
	var wordlist = make([]string, 0)
	for i := 0; i < 24; i++ {
		wordlist = append(wordlist, babbler.Babble())
	}

	fi, err := os.OpenFile("test-seq.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
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

	replacer, err := NewReplacer("test-seq.txt")
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
	babbler.Count = 6
	babbler.Separator = "-"
	for i := 0; i < 256; i++ {
		wordlist = append(wordlist, babbler.Babble())
	}
	rand.Seed(time.Now().UnixNano())
	if len(wordlist) == 0 {
		t.Fatal(fmt.Errorf("wordlist cannot have a len() of 0"))
	}
	for i := 0; i < 1*units.GiB/4; {
		s := wordlist[rand.Intn(len(wordlist))]
		if _, err := fi.WriteString(s); err != nil {
			t.Fatal(err.Error())
		}
		i += len(s)
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
	replaced, err := replacer.ReplaceChained()
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Printf("[gosed] --> replaced %d occurrences in %s\n", replaced, time.Since(start))
	var sedPath string
	if runtime.GOOS == "darwin" {
		sedPath = "gsed"
	} else {
		sedPath = "sed"
	}
	var args = fmt.Sprintf("#!/bin/bash\n%s -i '", sedPath)
	for i, v := range wordlist {
		if i != len(wordlist)-1 {
			args = fmt.Sprintf("%ss/%s/REPLACED-%d/g; ", args, v, i)
		} else if i == len(wordlist)-1 {
			args = fmt.Sprintf("%ss/%s/REPLACED-%d/g;' test-sed.txt", args, v, i)
		}
	}
	if err := ioutil.WriteFile("./sed.sh", []byte(args), 0777); err != nil {
		t.Fatal(err.Error())
	}
	start = time.Now()
	cmd := exec.Command("./sed.sh")
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

func TestFullSequential(t *testing.T) {
	defer Cleanup()
	babbler := babble.NewBabbler()
	var wordlist []string
	fi, err := os.OpenFile("template.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
	babbler.Count = 6
	babbler.Separator = "-"
	for i := 0; i < 256; i++ {
		wordlist = append(wordlist, babbler.Babble())
	}
	rand.Seed(time.Now().UnixNano())
	if len(wordlist) == 0 {
		t.Fatal(fmt.Errorf("wordlist cannot have a len() of 0"))
	}
	for i := 0; i < 1*units.GiB/4; {
		s := wordlist[rand.Intn(len(wordlist))]
		if _, err := fi.WriteString(s); err != nil {
			t.Fatal(err.Error())
		}
		i += len(s)
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
	fi2, err := os.OpenFile("test-sed-seq.txt", os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
	fi3, err := os.OpenFile("test-gosed-seq.txt", os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		t.Fatal(err.Error())
	}
	if _, err = io.Copy(io.MultiWriter(fi2, fi3), fi); err != nil {
		t.Fatal(err.Error())
	}
	replacer, err := NewReplacer("test-gosed-seq.txt")
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
	var sedPath string
	if runtime.GOOS == "darwin" {
		sedPath = "gsed"
	} else {
		sedPath = "sed"
	}
	start = time.Now()
	for i, v := range wordlist {
		out, err := exec.Command(sedPath, "-i", fmt.Sprintf("s/%s/REPLACED-%d/g", v, i), "test-sed-seq.txt").CombinedOutput()
		if err != nil {
			log.Println(string(out))
			t.Fatal(err.Error())
		}
	}
	log.Printf("[gnused] --> replaced all occurrences in %s\n", time.Since(start))
	log.Println("Comparing files...")
	hasher1 := sha256.New()
	hasher2 := sha256.New()
	hashReader1, err := os.Open("test-gosed-seq.txt")
	if err != nil {
		log.Printf("Error opening gosed test file: %s\n", err.Error())
		t.Fatal(err.Error())
	}
	hashReader2, err := os.Open("test-sed-seq.txt")
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
	fmt.Printf("%x:test-gosed-seq.txt\n%x:test-sed-seq.txt\n", hasher1.Sum(nil), hasher2.Sum(nil))
	if string(hasher1.Sum(nil)) != string(hasher2.Sum(nil)) {
		fmt.Printf("%x:test-gosed-seq.txt\n%x:test-sed-seq.txt", hasher1.Sum(nil), hasher2.Sum(nil))
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

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer func(in *os.File) {
		_ = in.Close()
	}(in)
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}