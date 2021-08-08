package main

import (
	"log"
	"os"
	"time"
)

func main() {
	// Creates a new replacer type with the provided file
	replacer, err := NewReplacer(os.Args[1])
	if err != nil {
		log.Fatal(err.Error())
	}
	// Creates a new old:new string mapping
	if err := replacer.NewStringMapping(os.Args[2], os.Args[3]); err != nil {
		log.Fatal(err.Error())
	}

	// Replace() Executes a SEQUENTIAL replace operation, meaning a temporary file is allocated for each
	// old:new mapping (slower, less CPU intensive)

	// Keep in mind this iterates through the mappings in order, so newly replaced byte sequences can
	// potentially be replaced by the next old:new mapping, but only if they match.
	start := time.Now()
	if _, err := replacer.Replace(); err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Operation completed in %s", time.Since(start))
}
