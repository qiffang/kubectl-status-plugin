package main

import (
	"fmt"
	"github.com/qiffang/kubectl-status-plugin/pkg/plugin"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func main()  {
	cmdutil.CheckErr(plugin.RunPlugin())
	fmt.Println("Done.")
	
}
