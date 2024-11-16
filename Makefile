run:
	@go run *.go

dev:
	@task start -vvv

test:
	@gotest ./... -v

tools:
	@command -v go &> /dev/null || (echo "Please install GoLang" && false)
	@command -v gotest &> /dev/null || (echo "Please install GoTest" && false)
	@command -v pkgx &> /dev/null || (echo "Please install PkgX" && false)
	@command -v task &> /dev/null || (echo "Please install Taskfile (or execute env +task)" && false)
	@command -v task &> /dev/null || (echo "Please install Overmind (or execute env +overmind)" && false)
	@command -v air &> /dev/null || (echo "Please install Air (or execute env +air)" && false)

