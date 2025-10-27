package main

import (
	"github.com/FangcunMount/iam-contracts/internal/apiserver"
)

func main() {
	apiserver.NewApp("iam-apiserver").Run()
}
