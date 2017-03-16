pkgname=shadowd
pkgver=22.39f9d22
pkgrel=1
pkgdesc="Secure login distribution service"
url="https://github.com/reconquest/shadowd"
arch=('i686' 'x86_64')
license=('GPL')
makedepends=('go')

source=(
    "git://github.com/reconquest/shadowd.git#branch=${BRANCH:-master}"
    "shadowd.service"
)
md5sums=('SKIP' 'SKIP')
backup=()

pkgver() {
    cd "${pkgname}"
    echo $(git rev-list --count master).$(git rev-parse --short master)
}

build() {
    cd "$srcdir/$pkgname"

    rm -rf "$srcdir/.go/src"

    mkdir -p "$srcdir/.go/src"

    export GOPATH=$srcdir/.go

    mv "$srcdir/$pkgname" "$srcdir/.go/src/"

    cd "$srcdir/.go/src/shadowd/"
    ln -sf "$srcdir/.go/src/shadowd/" "$srcdir/$pkgname"

    go get
}

package() {
    mkdir -p "$pkgdir/usr/bin"

    mkdir -p "$pkgdir/var/shadowd/ht/"
    mkdir -p "$pkgdir/var/shadowd/cert/"

    chmod 0600 "$pkgdir/var/shadowd/ht/"
    chmod 0600 "$pkgdir/var/shadowd/cert/"

    mkdir -p "$pkgdir/etc/systemd/system/"
    cp "$srcdir/.go/bin/$pkgname" "$pkgdir/usr/bin"
    cp "$srcdir/shadowd.service" "$pkgdir/etc/systemd/system/"
}
