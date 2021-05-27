module github.com/qiffang/kubectl-status-plugin

go 1.15

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.12.0
	github.com/moby/sys/mountinfo v0.4.1
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/cli-runtime v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/kubectl v0.21.1
	k8s.io/metrics v0.21.1
)
