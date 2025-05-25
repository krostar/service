{lib, ...}: {
  ci = {
    testers.go.enable = true;
    linters = {
      editorconfig-checker.settings.Exclude = ["./double/internal/*_generated*.go"];
      golangci-lint = {
        enable = true;
        linters = {
          exclusions = {
            paths = ["internal/example"];
            rules = [
              {
                path = "cfg/source/env/|cfg/source/flag/";
                linters = ["depguard"];
                text = "import 'reflect' is not allowed";
              }
              {
                path = "double/internal/generator/";
                linters = ["errcheck" "gosec" "goconst"];
                text = "Error return value of `file.Close` is not checked|Potential file inclusion via variable|make it a constant";
              }
            ];
          };

          settings = {
            iface.enable = lib.mkForce ["identical"];
            importas.alias = [
              {
                pkg = "github.com/krostar/cli/mapper/internal";
                alias = "mapper";
              }
              {
                pkg = "github.com/krostar/cli/cfg";
                alias = "clicfg";
              }
            ];
          };
        };
      };
    };
  };
}
