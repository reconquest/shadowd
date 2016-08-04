if ! which mongod &>/dev/null; then
    echo "fatal: dependency mongod is missing"
    exit 1
fi

tests:clone ../shadowd.test bin

_mongod="127.0.0.1:64999"
_shadowd_args=""

:shadowd-prepare() {
    tests:make-tmp-dir tables
    tests:ensure chmod 0700 tables
    tests:make-tmp-dir ssh
    tests:make-tmp-dir certs
}

:shadowd() {
    :shadowd-prepare

    tests:eval shadowd.test \
        --tables $(tests:get-tmp-dir)/tables/ \
        --keys $(tests:get-tmp-dir)/ssh/ \
        --certs $(tests:get-tmp-dir)/certs/ \
        ${_shadowd_args[@]} "$@"

    cat $(tests:get-stdout-file)
    cat $(tests:get-stderr-file) >&2

    return $(tests:get-exitcode)
}

:shadowd-listen() {
    :shadowd-prepare

    tests:ensure :shadowd -C --bytes 1024

    tests:run-background _shadowd shadowd.test \
        --tables $(tests:get-tmp-dir)/tables/ \
        --keys $(tests:get-tmp-dir)/ssh/ \
        --certs $(tests:get-tmp-dir)/certs/ \
        ${_shadowd_args[@]} -L "$@"
}

:mongod() {
    tests:make-tmp-dir db
    tests:run-background mongod_background \
        mongod --dbpath $(tests:get-tmp-dir)/db --port 64999
}

:shadowd-mongodb-config() {
    tests:put config <<CONF
[backend]
use = "mongodb"
dsn = "mongodb://$_mongod/shadowd"
CONF
    _shadowd_args=("--config" "$(tests:get-tmp-dir)/config")
}

:mongo() {
    tests:eval mongo --quiet $_mongod/shadowd --eval "$@"

    cat $(tests:get-stdout-file)
    cat $(tests:get-stderr-file) >&2

    return $(tests:get-exitcode)
}
