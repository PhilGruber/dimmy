#!/bin/bash
# Exercise package contents and postinst in a temporary installation root.
set -euo pipefail
trap 'echo "Debian install test failed at line $LINENO" >&2' ERR

REPO=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
TEST_DIR=$(mktemp -d /tmp/dimmy-deb-test.XXXXXXXX)
trap 'rm -rf -- "$TEST_DIR"' EXIT
BUILD="$TEST_DIR/build"
ROOT="$TEST_DIR/root"
CONFIGS=(dimmyd.conf.yaml rules.conf.yaml dimmy.conf)

mkdir -p "$BUILD/html" "$ROOT" "$TEST_DIR/bin"
cp "$REPO/Makefile" "$REPO/deb.control" "$BUILD/"
cp -R "$REPO/system" "$BUILD/system"
for name in "${CONFIGS[@]}"; do
    cp "$REPO/$name.example" "$BUILD/"
done
printf 'test page' > "$BUILD/html/index.html"
# Packaging does not depend on binary contents; skip compilation.
for name in dimmy dimmyd; do
    printf 'test binary' > "$BUILD/$name"
done
if ! make -C "$BUILD" -o all deb VERSION=0.0.0-test ARCH=all > "$TEST_DIR/build.log" 2>&1; then
    cat "$TEST_DIR/build.log" >&2
    exit 1
fi

unpack() {
    dpkg-deb -x "$BUILD/deb/dimmy.deb" "$ROOT"
}

unpack
shopt -s nullglob dotglob
for directory in "$ROOT/etc/dimmy" "$ROOT/var/lib/dimmy"; do
    entries=("$directory"/*)
    [[ ${#entries[@]} -eq 0 ]]
done
dpkg-deb -e "$BUILD/deb/dimmy.deb" "$TEST_DIR/control"
printf '/etc/init.d/dimmyd\n' > "$TEST_DIR/expected-conffiles"
cmp "$TEST_DIR/expected-conffiles" "$TEST_DIR/control/conffiles"

# Redirect install paths and stub account/ownership commands.
sed -e "s|/etc/dimmy|$ROOT/etc/dimmy|g" \
    -e "s|/usr/share/dimmy|$ROOT/usr/share/dimmy|g" \
    -e "s|/var/lib/dimmy|$ROOT/var/lib/dimmy|g" \
    "$TEST_DIR/control/postinst" > "$TEST_DIR/postinst"
for name in getent chown; do
    printf '#!/bin/sh\nexit 0\n' > "$TEST_DIR/bin/$name"
    chmod 755 "$TEST_DIR/bin/$name"
done

configure() {
    PATH="$TEST_DIR/bin:$PATH" bash "$TEST_DIR/postinst" configure
}

configure
for name in "${CONFIGS[@]}"; do
    cmp "$ROOT/etc/dimmy/$name" "$REPO/$name.example"
done

CONFIG="$ROOT/etc/dimmy"
DATABASE="$ROOT/var/lib/dimmy"
printf 'custom configuration\n' > "$CONFIG/dimmyd.conf.yaml"
: > "$CONFIG/rules.conf.yaml"
rm "$CONFIG/dimmy.conf"
ln -s missing-local-config "$CONFIG/dimmy.conf"
# Local examples from older packages must also survive postinst.
printf 'local example' > "$CONFIG/dimmy.conf.example"
for name in dimmy.db dimmy.db-wal dimmy.db-shm; do
    printf 'existing data\000\377' > "$DATABASE/$name"
done
cp -a "$CONFIG" "$TEST_DIR/expected-config"
cp -a "$DATABASE" "$TEST_DIR/expected-database"

for iteration in 1 2; do
    unpack
    configure
    for name in dimmyd.conf.yaml rules.conf.yaml dimmy.conf.example; do
        cmp "$CONFIG/$name" "$TEST_DIR/expected-config/$name"
    done
    for name in dimmy.db dimmy.db-wal dimmy.db-shm; do
        cmp "$DATABASE/$name" "$TEST_DIR/expected-database/$name"
    done
    [[ -L "$CONFIG/dimmy.conf" ]]
    [[ $(readlink "$CONFIG/dimmy.conf") == missing-local-config ]]
    [[ ! -e "$CONFIG/missing-local-config" ]]
done

echo 'Debian install test passed (fresh install and repeated upgrades).'
