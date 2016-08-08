:shadowd-listen "127.0.0.1:60002"

tests:ensure ssh-keygen -t rsa -b 1024 -f id_rsa
tests:ensure :shadowd -K blah/token '<' id_rsa.pub

tests:ensure curl -k "https://127.0.0.1:60002/ssh/blah/token"
tests:assert-no-diff stdout < id_rsa.pub
