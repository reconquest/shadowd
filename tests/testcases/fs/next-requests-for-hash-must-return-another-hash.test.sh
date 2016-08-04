:shadowd-listen "127.0.0.1:60002"

tests:ensure :shadowd -G --no-confirm --length 100 a/b/c/d '<<<' 'password'

tests:ensure curl -k "https://127.0.0.1:60002/t/a/b/c/d"
tests:assert-stdout-re '^.{63}$'

tests:value secret cat $(tests:get-stdout-file)

tests:describe "secret: $secret"

tests:ensure curl -k "https://127.0.0.1:60002/t/a/b/c/d"
tests:not tests:assert-stdout "$secret"

tests:value stub cat $(tests:get-stdout-file)

tests:ensure curl -k "https://127.0.0.1:60002/t/a/b/c/d"
tests:assert-no-diff stdout <<< "$stub"
