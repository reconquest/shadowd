tests:clone ../shadowd.test bin

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
        --certs $(tests:get-tmp-dir)/certs/ "$@"

    cat $(tests:get-stdout-file)
    cat $(tests:get-stderr-file) >&2

    return $(tests:get-exitcode)
}

:shadowd-listen() {
    :shadowd-prepare

    tests:run-background _shadowd shadowd.test \
        --tables $(tests:get-tmp-dir)/tables/ \
        --keys $(tests:get-tmp-dir)/ssh/ \
        --certs $(tests:get-tmp-dir)/certs/ \
        -L "$@"
}
