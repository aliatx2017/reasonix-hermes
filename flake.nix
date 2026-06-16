{
  description = "Reasonix Hermes — AI coding agent with MCP bridges, Discord bot, skills hub, and community tooling";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          # Main CLI
          reasonix = pkgs.buildGoModule {
            pname = "reasonix";
            version = "1.8.2";
            src = ./.;
            vendorHash = null; # proxy vendor mode — uses go.sum for integrity; fully reproducible on nixos-unstable
            subPackages = [ "cmd/reasonix" ];
            ldflags = [
              "-s" "-w"
              "-X reasonix/internal/version.Version=${self.shortRev or "dev"}"
            ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes AI coding agent CLI";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
              mainProgram = "reasonix";
            };
          };

          # MCP bridge server (6 tools: run, doctor, plan, orchestrate, get_skill, get_skills)
          reasonix-mcpbridge = pkgs.buildGoModule {
            pname = "reasonix-mcpbridge";
            version = "1.8.2";
            src = ./.;
            vendorHash = null; # proxy vendor mode — uses go.sum for integrity; fully reproducible on nixos-unstable
            subPackages = [ "cmd/reasonix-mcpbridge" ];
            ldflags = [ "-s" "-w" ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes MCP bridge server";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
              mainProgram = "reasonix-mcpbridge";
            };
          };

          # Hindsight memory server (3 tools: retain, recall, reflect; SQLite + vector)
          reasonix-memoryserver = pkgs.buildGoModule {
            pname = "reasonix-memoryserver";
            version = "1.8.2";
            src = ./.;
            vendorHash = null; # proxy vendor mode — uses go.sum for integrity; fully reproducible on nixos-unstable
            subPackages = [ "cmd/reasonix-memoryserver" ];
            ldflags = [ "-s" "-w" ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes Hindsight memory MCP server";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
              mainProgram = "reasonix-memoryserver";
            };
          };

          # Native Go hook runner
          reasonix-hooks = pkgs.buildGoModule {
            pname = "reasonix-hooks";
            version = "1.8.2";
            src = ./.;
            vendorHash = null; # proxy vendor mode — uses go.sum for integrity; fully reproducible on nixos-unstable
            subPackages = [ "cmd/reasonix-hooks" ];
            ldflags = [ "-s" "-w" ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes native Go hook runner";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
              mainProgram = "reasonix-hooks";
            };
          };

          # Discord bot
          reasonix-bot = pkgs.buildGoModule {
            pname = "reasonix-bot";
            version = "1.8.2";
            src = ./.;
            vendorHash = null; # proxy vendor mode — uses go.sum for integrity; fully reproducible on nixos-unstable
            subPackages = [ "bot" ];
            ldflags = [ "-s" "-w" ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes Discord bot";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
              mainProgram = "reasonix-bot";
            };
          };

          # PR review CLI for GitHub Actions
          reasonix-pr-review = pkgs.buildGoModule {
            pname = "reasonix-pr-review";
            version = "1.8.2";
            src = ./.;
            vendorHash = null;
            subPackages = [ "cmd/reasonix-pr-review" ];
            ldflags = [ "-s" "-w" ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes PR review CLI for GitHub Actions";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
              mainProgram = "reasonix-pr-review";
            };
          };

          # E2E benchmark tool
          reasonix-e2ebench = pkgs.buildGoModule {
            pname = "reasonix-e2ebench";
            version = "1.8.2";
            src = ./.;
            vendorHash = null;
            subPackages = [ "cmd/e2ebench" ];
            ldflags = [ "-s" "-w" ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes e2e benchmarking tool";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
              mainProgram = "e2ebench";
            };
          };

          # All-in-one metapackage
          reasonix-full = pkgs.symlinkJoin {
            name = "reasonix-full";
            paths = with self.packages.${system}; [
              reasonix
              reasonix-mcpbridge
              reasonix-memoryserver
              reasonix-hooks
              reasonix-bot
              reasonix-pr-review
              reasonix-e2ebench
            ];
            meta = with pkgs.lib; {
              description = "Reasonix Hermes — all binaries";
              homepage = "https://github.com/aliatx2017/reasonix-hermes";
              license = licenses.mit;
            };
          };

          default = self.packages.${system}.reasonix;
        };

        # Development shell with Go + required tooling
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_25
            gopls
            golangci-lint
            nodejs_22
            pnpm
          ];
          shellHook = ''
            echo "Reasonix Hermes dev shell — Go $(go version | awk '{print $3}')"
          '';
        };

        apps = {
          reasonix = flake-utils.lib.mkApp { drv = self.packages.${system}.reasonix; };
          reasonix-mcpbridge = flake-utils.lib.mkApp { drv = self.packages.${system}.reasonix-mcpbridge; };
          reasonix-memoryserver = flake-utils.lib.mkApp { drv = self.packages.${system}.reasonix-memoryserver; };
          reasonix-hooks = flake-utils.lib.mkApp { drv = self.packages.${system}.reasonix-hooks; };
          reasonix-bot = flake-utils.lib.mkApp { drv = self.packages.${system}.reasonix-bot; };
          reasonix-pr-review = flake-utils.lib.mkApp { drv = self.packages.${system}.reasonix-pr-review; };
          reasonix-e2ebench = flake-utils.lib.mkApp { drv = self.packages.${system}.reasonix-e2ebench; };
          default = self.apps.${system}.reasonix;
        };
      }
    );
}
