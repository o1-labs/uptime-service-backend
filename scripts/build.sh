#!/usr/bin/env bash

set -e

if [[ "$OUT" == "" ]]; then
  OUT="$PWD/result"
fi

ref_signer="$PWD/external/c-reference-signer"

mkdir -p "$OUT"/{headers,bin}
rm -f "$OUT"/libmina_signer.so # Otherwise re-building without clean causes permissions issue
if [[ "$LIB_MINA_SIGNER" == "" ]]; then
  # No nix
  cp -R "$ref_signer" "$OUT"
  make -C "$OUT/c-reference-signer" clean libmina_signer.so
  cp "$OUT/c-reference-signer/libmina_signer.so" "$OUT"
else
  cp "$LIB_MINA_SIGNER" "$OUT"/libmina_signer.so
fi
cp "$ref_signer"/*.h "$OUT/headers"

case "$1" in
  db-migrate-up)
    cd src/cmd/db_migration
    $GO run main.go up
    ;;
  db-migrate-down)
    cd src/cmd/db_migration
    $GO run main.go down
    ;;
  test)
    cd src/delegation_backend
    LD_LIBRARY_PATH="$OUT" $GO test
    ;;
  integration-test)
    cd src/integration_tests
    $GO test -v --timeout 30m
    ;;
  docker)
    if [[ "$TAG" == "" ]]; then
      echo "Specify TAG env variable."
      exit 1
    fi
    # set image name to 673156464838.dkr.ecr.us-west-2.amazonaws.com/uptime-service-backend if IMAGE_NAME is not set
    IMAGE_NAME=${IMAGE_NAME:-uptime-service-backend}
    docker build -t "$IMAGE_NAME:$TAG" -f dockerfiles/Dockerfile-delegation-backend .
    ;;
  "")
    cd src/cmd/delegation_backend
    $GO build -o "$OUT/bin/delegation_backend"
    echo "to run use cmd: LD_LIBRARY_PATH=result ./result/bin/delegation_backend"
    ;;
  *)
    echo "unknown command $1"
    exit 2
    ;;
esac
