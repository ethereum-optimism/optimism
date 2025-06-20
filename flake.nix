{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    foundry.url = "github:shazow/foundry.nix/main";
  };

  outputs =
    inputs:
    inputs.flake-utils.lib.eachDefaultSystem (
      system:
      let
        overlays = [
          inputs.foundry.overlay
        ];
        pkgs = import inputs.nixpkgs { inherit overlays system; };

        go_1_22_7 = pkgs.go_1_22.overrideAttrs (oldAttrs: {
          version = "1.22.7";

          src = pkgs.fetchurl {
            url = "https://go.dev/dl/go1.22.7.src.tar.gz";
            sha256 = "sha256-ZkMth9heDPrD7f/mN9WTD8Td9XkzE/4R5KDzMwI8h58=";
          };
        });

        espressoGoLibFile = pkgs.stdenv.mkDerivation rec {
          pname = "libespresso_crypto_helper";
          version = "0.2.1";

          baseUrl = "https://github.com/EspressoSystems/espresso-network/releases/download/sdks%2Fgo%2Fv${version}";
          source =
            {
              "x86_64-linux" = pkgs.fetchurl {
                url = baseUrl + "/libespresso_crypto_helper-x86_64-unknown-linux-gnu.so";
                sha256 = "sha256:b3e28f7dc755d72b27a2a43c2bcfdc0e4e82096e03596a01447bd8f406e6653c";
              };
              "x86_64-darwin" = pkgs.fetchurl {
                url = baseUrl + "/libespresso_crypto_helper-x86_64-apple-darwin.dylib";
                sha256 = "sha256:716cb9eb548222ed1c7b5d1585bd5f03d0680cbae3f8db14cbf37837f54b9788";
              };
              "aarch64-linux" = pkgs.fetchurl {
                url = baseUrl + "/libespresso_crypto_helper-aarch64-unknown-linux-gnu.so";
                sha256 = "sha256:886aef8aeaa0d5695abc6a9ae54f085899a031371c10755218e387442ecb331f";
              };
              "aarch64-darwin" = pkgs.fetchurl {
                url = baseUrl + "/libespresso_crypto_helper-aarch64-apple-darwin.dylib";
                sha256 = "sha256:6c74ec631ccd9d23258ff99a8060068a548740fac814633ceab2ad7c7dc90a74";
              };
            }
            ."${system}";

          dontUnpack = true;
          installPhase = ''
            mkdir -p $out/lib
            cp ${source} $out/lib/
          '';
        };

        eth-beacon-genesis = pkgs.buildGoModule rec {
          pname = "eth-beacon-genesis";
          version = "703e97a";

          src = pkgs.fetchFromGitHub {
            owner = "ethpandaops";
            repo = pname;
            rev = version;
            hash = "sha256-Toal70A8cnIAtR4iCacRQ5vT+MHUlMc81l1dzjj56mA=";
          };

          vendorHash = "sha256-keBJHjl42o6guAAAWoESJateXVG3hotdSnDv2pf1Lv4=";
          proxyVendor = true;

          doCheck = false;
        };

        eth2-val-tools = pkgs.buildGoModule rec {
          pname = "eth2-val-tools";
          version = "662955e";

          src = pkgs.fetchFromGitHub {
            owner = "protolambda";
            repo = pname;
            rev = version;
            hash = "sha256-UpQmCS/FrY667EnNH2XCTJhzhLOpsfS5GUhGvXGG65U=";
          };

          vendorHash = "sha256-IblAuZgk7EBkfcFoEugzb9pO454zfHq6RxIfgvUFBDo=";
          proxyVendor = true;

          doCheck = false;
        };

        enclaver = pkgs.rustPlatform.buildRustPackage rec {
          pname = "enclaver";
          version = "0.5.0";

          src = pkgs.fetchFromGitHub {
            owner = "enclaver-io";
            repo = pname;
            rev = "v${version}";
            hash = "sha256-gfzfgcnVDRqywAJ/SC2Af6VfHPELDkoVlkhaKElMP2g=";
          };

          useFetchCargoVendor = true;
          cargoHash = "sha256-o+CzTn5++Mj6SP9yFeTOBn4feapnL2m1EsYmXQBqTuc=";
          cargoRoot = "enclaver";
          buildAndTestSubdir = cargoRoot;
        };

      in
      {
        formatter = pkgs.nixfmt-rfc-style;

        devShells = {
          default = pkgs.mkShell {
            buildInputs = [
              pkgs.zlib
              espressoGoLibFile
            ];

            packages = [
              enclaver
              eth-beacon-genesis
              eth2-val-tools
              go_1_22_7

              pkgs.awscli2
              pkgs.cargo
              pkgs.dasel
              pkgs.foundry-bin
              pkgs.go-ethereum
              pkgs.jq
              pkgs.just
              pkgs.just
              pkgs.pnpm
              pkgs.python311
              pkgs.shellcheck
              pkgs.uv
              pkgs.yq-go
            ];

            shellHook = ''
              export FOUNDRY_DISABLE_NIGHTLY_WARNING=1
              export MACOSX_DEPLOYMENT_TARGET=14.5
              export PATH=$PATH:$PWD/op-deployer/bin
            '';
          };
        };
      }
    );
}
