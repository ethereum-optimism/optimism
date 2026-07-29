{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/24.11";
    utils.url = "github:numtide/flake-utils";
    crane.url = "github:ipetkov/crane";

    fenix = {
      url = "github:nix-community/fenix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      nixpkgs,
      utils,
      crane,
      fenix,
      ...
    }:
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        # A useful helper for folding a list of `prevSet -> newSet` functions
        # into an attribute set.
        composeAttrOverrides = defaultAttrs: overrides: builtins.foldl'
            (acc: f: acc // (f acc)) defaultAttrs overrides;

        cargoTarget = pkgs.stdenv.hostPlatform.rust.rustcTargetSpec;
        cargoTargetEnvVar = builtins.replaceStrings ["-"] ["_"]
            (pkgs.lib.toUpper cargoTarget);

        cargoTOML = builtins.fromTOML (builtins.readFile ./Cargo.toml);
        packageVersion = cargoTOML.workspace.package.version;

        rustStable = fenix.packages.${system}.stable.withComponents [
          "cargo" "rustc" "rust-src" "clippy"
        ];

        rustNightly = fenix.packages.${system}.latest;

        craneLib = (crane.mkLib pkgs).overrideToolchain rustStable;

        nativeBuildInputs = [
          pkgs.pkg-config
          pkgs.libgit2
          pkgs.perl
        ];

        withClang = prev: {
          buildInputs = prev.buildInputs or [] ++ [
            pkgs.clang
          ];
          LIBCLANG_PATH = "${pkgs.libclang.lib}/lib";
        };

        withMaxPerf = prev: {
          cargoBuildCommand = "cargo build --profile=maxperf";
          cargoExtraArgs = prev.cargoExtraArgs or "" + " --features=jemalloc,asm-keccak";
          RUSTFLAGS = prev.RUSTFLAGS or [] ++ [
            "-Ctarget-cpu=native"
          ];
        };

        withMold = prev: {
          buildInputs = prev.buildInputs or [] ++ [
            pkgs.mold
          ];
          "CARGO_TARGET_${cargoTargetEnvVar}_LINKER" = "${pkgs.llvmPackages.clangUseLLVM}/bin/clang";
          RUSTFLAGS = prev.RUSTFLAGS or [] ++ [
            "-Clink-arg=-fuse-ld=${pkgs.mold}/bin/mold"
          ];
        };

        withOp = prev: {
          cargoExtraArgs = prev.cargoExtraArgs or "" + " -p op-reth --bin=op-reth";
        };

        mkReth = overrides: craneLib.buildPackage (composeAttrOverrides {
          pname = "reth";
          version = packageVersion;
          src = ./.;
          inherit nativeBuildInputs;
          doCheck = false;
        } overrides);

      in
      {
        packages = rec {

          reth = mkReth ([
            withClang
            withMaxPerf
          ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
            withMold
          ]);

          op-reth = mkReth ([
            withClang
            withMaxPerf
            withOp
          ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
            withMold
          ]);

          default = reth;
        };

        devShell = let
          overrides = [
            withClang
          ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
            withMold
          ];
        in craneLib.devShell (composeAttrOverrides {
          packages = nativeBuildInputs ++ [
            rustNightly.rust-analyzer
            rustNightly.rustfmt
            pkgs.cargo-nextest
          ];

          # Remove the hardening added by nix to fix jmalloc compilation error.
          # More info: https://github.com/tikv/jemallocator/issues/108
          hardeningDisable = [ "fortify" ];

        } overrides);
      }
    );
}
