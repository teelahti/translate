{
  description = "CLI translation tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = rec {
          translate = pkgs.buildGoModule {
            pname = "translate";
            version = "2.0.0";
            src = ./.;
            # Use nixpkgs.lib.fakeHash until the real one is calculated by nix build
            # vendorHash = nixpkgs.lib.fakeHash;
            vendorHash = "sha256-7yUhdufFli8fuNZgWxiiYvk7CdiToTa7ou6FACXreNA=";
            buildInputs = [ ];
          };

          default = translate;
        };

        devShells = {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gotools
              pkgs.gopls
              pkgs.golangci-lint
            ];
          };
        };
      }
    );
}
