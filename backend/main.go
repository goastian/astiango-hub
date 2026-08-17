package main

import (
	"github.com/goastian/astiango-hub/core/cmd"
	"github.com/goastian/astiango-hub/core/config"
	"github.com/goastian/astiango-hub/core/utils"
)

func init() {
	config.InitConfig()
}

func main() {
	go func() {
		err := cmd.Execute()
		if err != nil {
			panic(err)
		}
	}()
	utils.DefaultWait()
}
