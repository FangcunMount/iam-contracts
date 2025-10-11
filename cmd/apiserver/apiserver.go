package main

import (
	"github.com/fangcun-mount/iam-contracts/internal/apiserver"
)

func main() {
	apiserver.NewApp("web-framework").Run()
}
