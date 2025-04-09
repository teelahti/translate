{
  description = "CLI translation tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs?tag=24.11";
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
            version = "1.0.0";
            src = ./.;
            # Use nixpkgs.lib.fakeHash until the real one is calculated by nix build
            # vendorHash = nixpkgs.lib.fakeHash;
            vendorHash = "sha256-NCyjZsSpMUFBV2wTFfmpcu4JKWZVb5gZ0qFCSQEO2mI=";
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
