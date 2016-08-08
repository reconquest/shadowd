tests:ensure ssh-keygen -t rsa -b 1024 -f id_rsa
tests:ensure :shadowd -K blah/token '<' id_rsa.pub

tests:ensure ssh-keygen -t rsa -b 1024 -f id_rsa_2
tests:ensure :shadowd --truncate -K blah/token '<' id_rsa_2.pub

tests:assert-no-diff $(tests:get-tmp-dir)/ssh/blah/token <<KEYS
$(cat id_rsa_2.pub)
KEYS
