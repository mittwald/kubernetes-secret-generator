{
  description = "A Nix-flake-based Go 1.22 development environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-25.11";
    systems.url = "github:nix-systems/default";
    pre-commit-hooks.url = "github:cachix/git-hooks.nix";
  };

  outputs =
    inputs@{
      self,
      nixpkgs,
      systems,
      ...
    }:
    let
      supportedSystems = import systems;
      forEachSystem =
        f:
        nixpkgs.lib.genAttrs supportedSystems (
          system:
          f (
            import nixpkgs {
              inherit system;
              overlays = [ self.overlays.default ];
            }
          )
        );
    in
    {

      overlays.default = final: prev: {
        operator-sdk-0_18_2 = final.buildGoModule rec {
          pname = "operator-sdk";
          version = "0.18.2";

          src = final.fetchFromGitHub {
            owner = "operator-framework";
            repo = "operator-sdk";
            tag = "v${version}";
            hash = "sha256-aI/TKFvh+GIDqQqtVkmMH5INooeDZJby9ol7Ahfufws=";
            #hash = final.lib.fakeHash;
          };

          vendorHash = "sha256-N8SEbL2Rf7MlgriQEhCTcOgHEdstvydZDpw1eE29q00=";

          nativeBuildInputs = [
            final.makeWrapper
          ];

          buildInputs = [
            final.go
          ];

          subPackages = [
            "cmd/operator-sdk"
          ];

          allowGoReference = true;

          postFixup = ''
            wrapProgram $out/bin/operator-sdk --prefix PATH : ${final.lib.makeBinPath [ final.go ]}
          '';
        };

      };

      devShells = forEachSystem (pkgs: {
        default =
          #let
          #  pkgs = nixpkgs.legacyPackages.${system};
          #  #pkgs-unstable = nixpkgs-unstable.legacyPackages.${system};
          #  #inherit (self.checks.${system}.pre-commit-check) shellHook enabledPackages;
          #in
          pkgs.mkShell {
            GOROOT = "${pkgs.go}/share/go";

            #shellHook = ''
            #  ${shellHook}

            #  export PATH="$PATH:$(${pkgs.go}/bin/go env GOPATH)/bin"
            #'';
            #buildInputs = enabledPackages;

            hardeningDisable = [ "fortify" ];

            packages = [
              pkgs.go
              pkgs.operator-sdk-0_18_2
              pkgs.kind
            ];
          };
      });
    };
}
