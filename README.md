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
  replacer, err := gosed.NewReplacer("hugeAssFile.txt");
  if err != nil {
      log.Fatal(err.Error())
  }
  if err := replacer.NewMapping("oldString", "newString"); err != nil {
      log.Fatal(err.Error())
  }
  if _, err := replacer.Replace(); err != nil {
      log.Fatal(err.Error())
  }
  
}
```
