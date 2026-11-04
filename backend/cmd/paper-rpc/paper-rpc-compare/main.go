package main

import (
	"fmt"

	"github.com/GouTouZhao/PaperCompare/pkg/common/cmd"
)

func main() {
	if err := cmd.NewRpcCompareCmd().Exec(); err != nil {
		fmt.Println(err)
	}
}

