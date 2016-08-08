:mongod
:shadowd-mongodb-config

:shadowd-listen "127.0.0.1:60002"

tests:ensure :shadowd -G --no-confirm --length 100 a/b/c/d '<<<' 'password'

tests:ensure curl -k "https://127.0.0.1:60002/t/a/b/c/d"
tests:assert-stdout-re '^.{63}$'
