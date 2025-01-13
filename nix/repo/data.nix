{lib, ...}: {
  ci.linters.golangci-lint.linters-settings = {
    importas.alias = lib.mkForce [
      {
        pkg = "github.com/krostar/service/net/tls";
        alias = "tlsnetservice";
      }
    ];
  };
}
