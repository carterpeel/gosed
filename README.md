# gosed
`sed -i` written in Golang as an importable module.
This is very useful for replacing specific strings of text in massive files, because sometimes ingesting `someBigAssFile.txt` into memory isn't a great idea.

# Usage
```go
package main

import (
  "github.com/carterpeel/gosed"
  "log"
)

func main() {
  // Must have os.O_RDWR file descriptor for proper functionality
  fi, err := os.OpenFile("./someBigAssFile.txt", os.O_RDWR, 0644)
  if err != nil {
    log.Fatalf(err.Error())
  }
  defer fi.Close()
  
  err = gosed.ReplaceIn(fi, []byte("oldString"), []byte("newString"))
  if err != nil {
    panic(err.Error())
  } 
}
```
