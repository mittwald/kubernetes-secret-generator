{
  inputs = { nixpkgs.url = "github:nixos/nixpkgs"; };

  outputs = { self, nixpkgs }:
    let
        pkgs = nixpkgs.legacyPackages.x86_64-linux;
        operator = pkgs.buildGoModule rec {
            pname = "operator-sdk";
            version = "0.18.2";

            src = pkgs.fetchFromGitHub {
                owner = "operator-framework";
                repo = pname;
                rev = "v${version}";
                sha256 = "sha256-aI/TKFvh+GIDqQqtVkmMH5INooeDZJby9ol7Ahfufws=";
            };

            vendorSha256 = "sha256-N8SEbL2Rf7MlgriQEhCTcOgHEdstvydZDpw1eE29q00=";

            doCheck = false;

            subPackages = [ "cmd/operator-sdk" ];

            nativeBuildInputs = [ pkgs.makeWrapper ];
            buildInputs = [ pkgs.go ];

            # operator-sdk uses the go compiler at runtime
            allowGoReference = true;
            postFixup = ''
                wrapProgram $out/bin/operator-sdk --prefix PATH : ${nixpkgs.lib.makeBinPath [ pkgs.go ]}
            '';

            meta = with nixpkgs.lib; {
                description = "SDK for building Kubernetes applications. Provides high level APIs, useful abstractions, and project scaffolding";
                homepage = "https://github.com/operator-framework/operator-sdk";
                license = licenses.asl20;
                maintainers = with maintainers; [ arnarg ];
                platforms = platforms.linux ++ platforms.darwin;
            };
        };

        deps =[pkgs.helm pkgs.kind pkgs.gnumake pkgs.kubectl operator];
    in {
      # packages.x86_64-linux.hello = pkgs.hello;
      packages.x86_64-linux.defaultPackage.x86_64-linux = operator;

      devShell.x86_64-linux =
        pkgs.mkShell { buildInputs = deps; };
   };
}
