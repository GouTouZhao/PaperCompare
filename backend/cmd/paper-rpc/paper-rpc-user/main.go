package main

import (
	"fmt"

	"github.com/GouTouZhao/PaperCompare/pkg/common/cmd"
)

func main() {
	if err := cmd.NewRpcUserCmd().Exec(); err != nil {
		fmt.Println(err)
	}
}

