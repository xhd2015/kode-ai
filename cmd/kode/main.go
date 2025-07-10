package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/kode-ai/run"
)

func main() {
	err := run.Main(os.Args[1:], run.Options{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
