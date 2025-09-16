{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
	buildInputs = [
		pkgs.go
		pkgs.golangci-lint
	];
	shellHook = ''
		# Install noctx if not already installed
		go install github.com/sonatard/noctx/cmd/noctx@latest

		# Add Go binaries to PATH
		export PATH=$PATH:$(go env GOPATH)/bin
	'';
}
