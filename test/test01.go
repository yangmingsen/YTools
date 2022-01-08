package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("cpu:", runtime.NumCPU())
}
