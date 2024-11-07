{
  description = "CLI translation tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs?tag=24.05";
  };

  outputs = { self, nixpkgs, ... }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        rec {
          translate = pkgs.buildGoModule {
            pname = "translate";
            version = "1.0.0";
            src = ./.;
            # Use nixpkgs.lib.fakeHash until the real one is calculated by nix build
            # vendorHash = nixpkgs.lib.fakeHash;
            vendorHash = "sha256-NCyjZsSpMUFBV2wTFfmpcu4JKWZVb5gZ0qFCSQEO2mI=";
            buildInputs = [];
          };

          default = translate;
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gotools
              pkgs.gopls
              pkgs.golangci-lint-langserver
            ];
          };
        });
    };
}
