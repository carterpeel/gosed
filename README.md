# gosed
`sed -i` written in Golang as an importable module.

# Usage
```go
package main

import (
  gosed "github.com/carterpeel/gosed"
  "log"
)

func main() {
  // Must have os.O_RDWR file descriptor for proper functionality
  fi, err := os.OpenFile("./someBigAssFile", os.O_RDWR, 0644)
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
