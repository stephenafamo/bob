{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
	buildInputs = [
		pkgs.go
		pkgs.golangci-lint
	];
	shellHook = ''
		go install github.com/sonatard/noctx/cmd/noctx@latest
		go install mvdan.cc/gofumpt@latest

		# Add Go binaries to PATH
		export PATH=$PATH:$(go env GOPATH)/bin
	'';
}
