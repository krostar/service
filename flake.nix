{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    harmony = {
      url = "github:krostar/harmony";
      inputs = {
        synergy.follows = "synergy";
        nixpkgs-unstable.follows = "nixpkgs";
      };
    };
    synergy = {
      url = "github:krostar/synergy";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = {synergy, ...} @ inputs:
    synergy.lib.mkFlake {
      inherit inputs;
      src = ./nix;
      eval.synergy.restrictDependenciesUnits.harmony = ["harmony"];
    };
}
