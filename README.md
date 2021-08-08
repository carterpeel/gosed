# gosed
`sed -i` written in Golang as an importable module.
This is very useful for replacing specific strings of text in massive files, because sometimes ingesting `someBigAssFile.txt` into memory isn't a great idea.

# Sequential Replacer Usage
```go
package main
import (
  "github.com/carterpeel/gosed"
  "log"
)
func main() {
  // Creates a new replacer type with the provided file 
  replacer, err := gosed.NewReplacer("hugeAssFile.txt");
  if err != nil {
    log.Fatal(err.Error())
  }
  // Creates a new old:new string mapping 
  if err := replacer.NewStringMapping("oldString", "newString"); err != nil {
    log.Fatal(err.Error())
  }
  // Creates a new old:new byte sequence mapping 
  if err := replacer.NewMapping([]byte("oldString2"), []byte("newString2")); err != nil {
    log.Fatal(err.Error())
  }
  
  
  // Replace() Executes a SEQUENTIAL replace operation, meaning a temporary file is allocated for each
  // old:new mapping (slower, less CPU intensive)
  
  // Keep in mind this iterates through the mappings in order, so newly replaced byte sequences can 
  // potentially be replaced by the next old:new mapping, but only if they match.
  if _, err := replacer.Replace(); err != nil {
    log.Fatal(err.Error())
  }
}
```
# Chained Replacer Usage
```go
package main
import (
  "github.com/carterpeel/gosed"
  "log"
)
func main() {
  // Creates a new replacer type with the provided file 
  replacer, err := gosed.NewReplacer("hugeAssFile.txt");
  if err != nil {
    log.Fatal(err.Error())
  }
  // Creates a new old:new string mapping 
  if err := replacer.NewStringMapping("oldString", "newString"); err != nil {
    log.Fatal(err.Error())
  }
  // Creates a new old:new byte sequence mapping 
  if err := replacer.NewMapping([]byte("oldString2"), []byte("newString2")); err != nil {
    log.Fatal(err.Error())
  }
  
  
  // Replace() Executes a CHAINED replace operation, meaning the readers are chained in order
  // and only need to allocate a single temporarily file. (faster, more CPU intensive)
  
  // Keep in mind this iterates through the mappings in order, so newly replaced byte sequences can 
  // potentially be replaced by the next old:new mapping, but only if they match.
  if _, err := replacer.ReplaceChained(); err != nil {
    log.Fatal(err.Error())
  }
}
```
