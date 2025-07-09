{
	inputs = {
		utils.url = "github:numtide/flake-utils";
		nixpkgs.url = "nixpkgs/nixos-unstable";
	};

	outputs = { self, nixpkgs, utils }:
		utils.lib.eachDefaultSystem (system:
			let
				pkgs = nixpkgs.legacyPackages."${system}";
			in rec {
				devShell = pkgs.mkShell {
					shellHook = ''
						export SHELL=${pkgs.fish}/bin/fish
					'';
					nativeBuildInputs = with pkgs; [
						go_1_24
						gopls
						golangci-lint
						golangci-lint-langserver
					];
				};
			}
		);
}
