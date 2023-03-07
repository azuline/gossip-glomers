{
  description = "gossip glomers development environment";

  inputs = {
    nixpkgs.url = github:nixos/nixpkgs/nixos-unstable;
    flake-utils.url = github:numtide/flake-utils;
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in rec {
        devShell = pkgs.mkShell {
          buildInputs = [
            (pkgs.buildEnv {
              name = "glomers dev shell";
              paths = with pkgs; [
                gnumake
                golangci-lint
                go_1_18
                gopls
                gopkgs
                go-outline
                delve
                gotools
              ];
            })
          ];
          shellHook = ''
            # Isolate build stuff to this repo's directory.

            export GLOMERS_ROOT="$(pwd)"
            export GLOMERS_CACHE_ROOT="$(pwd)/.cache"

            export GOCACHE="$GLOMERS_CACHE_ROOT/go/cache"
            export GOENV="$GLOMERS_CACHE_ROOT/go/env"
            export GOPATH="$GLOMERS_CACHE_ROOT/go/path"
            export GOMODCACHE="$GOPATH/pkg/mod"
            export GOROOT=
            export PATH=$(go env GOPATH)/bin:$PATH
          '';
        };
      }
    );
}
