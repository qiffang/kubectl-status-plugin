export GO111MODULE=on

generate: pkg/statik/statik.go

pkg/statik/statik.go: pkg/templates/templates.tmpl
	go get github.com/rakyll/statik@v0.1.7
	go generate ./pkg/... ./cmd/...
	# statik generates non-fmt compliant files, so we have an extra "go fmt" here
	go fmt pkg/statik/statik.go