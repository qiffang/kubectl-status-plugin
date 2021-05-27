package main

import (
	"fmt"
	"github.com/qiffang/kubectl-status-plugin/pkg/cluster-status"
)

func main()  {
	enableSeLinux := clusterstatus.SelinuxEnabled()
	fmt.Println(fmt.Sprintf("SelinuxEnabled: %v", enableSeLinux))

	disableSwap, err := clusterstatus.SwapDisabled()
	if err != nil {
		fmt.Println(fmt.Sprintf("detect swag failed,%v", err))
	}

	fmt.Println(fmt.Sprintf("SwapDisabled: %v", disableSwap))

	brNfCallIptables := clusterstatus.EnableBridgeNfCallIptables()
	fmt.Println(fmt.Sprintf("EnableBridgeNfCallIptables: %v", brNfCallIptables))
}