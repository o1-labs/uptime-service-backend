with import (fetchTarball "https://nixos.org/channels/nixos-unstable/nixexprs.tar.xz") { };
let
  minaSigner = import ./external/c-reference-signer;
in
{
  devEnv = stdenv.mkDerivation {
    name = "dev";
    buildInputs = [ stdenv go glibc minaSigner ];
    shellHook = ''
      export LIB_MINA_SIGNER=${minaSigner}/lib/libmina_signer.so
      return
    '';
  };
}
