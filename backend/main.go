package main

import (
	"github.com/goastian/astiango-hub/core/cmd"
	"github.com/goastian/astiango-hub/core/config"
	"github.com/goastian/astiango-hub/core/user"
	"github.com/goastian/astiango-hub/core/utils"
)

func init() {
	config.InitConfig()
	if err := user.ValidateJWTConfiguration(); err != nil {
		panic(err)
	}
	if err := utils.ValidateSecurityConfiguration(); err != nil {
		panic(err)
	}
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
