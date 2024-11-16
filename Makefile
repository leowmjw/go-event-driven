run:
	@go run *.go

task:
	@task dev -vvv

test:
	@gotest ./...

tools:
	@command -v go &> /dev/null || (echo "Please install GoLang" && false)
	@command -v gotest0 &> /dev/null || (echo "Please install GoTest" && false)
	@command -v pkgx &> /dev/null || (echo "Please install PkgX" && false)
	@command -v task &> /dev/null || (echo "Please install Taskfile" && false)
