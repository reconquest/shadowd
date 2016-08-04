:shadowd-listen 127.0.0.1:60002

tests:ensure \
    :shadowd --no-confirm --length 100 -G pool/token '<<<' "password"

tests:ensure \
    :shadowd --no-confirm --length 100 -G pool/token2 '<<<' "password"

tests:ensure \
    :shadowd --no-confirm --length 100 -G pool2/token3 '<<<' "password"

tests:ensure curl -k "https://127.0.0.1:60002/t/pool/"
tests:assert-no-diff stdout <<TOKENS
token
token2
TOKENS
