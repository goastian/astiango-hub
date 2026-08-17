package main

import (
	"github.com/goastian/astiango-hub/core/cmd"
	"github.com/goastian/astiango-hub/core/utils"
)

func main() {
	go func() {
		err := cmd.Execute()
		if err != nil {
			panic(err)
		}
	}()
	utils.DefaultWait()
}
