:shadowd-listen "127.0.0.1:60002"

tests:ensure :shadowd -G --no-confirm --length 100 a/b/c/d '<<<' 'old'

tests:ensure curl -X PUT -k "https://127.0.0.1:60002/t/a/b/c/d"

salts=$(cat $(tests:get-stdout-file))

tests:ensure curl -X PUT -k "https://127.0.0.1:60002/t/a/b/c/d"
tests:assert-no-diff stdout <<SALTS
$salts
SALTS

tests:ensure sort \| uniq \| wc -l <<< "$salts"
tests:assert-stdout-re '^10$'
