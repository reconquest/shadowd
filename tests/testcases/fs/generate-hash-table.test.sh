tests:ensure \
    :shadowd --no-confirm --length 100 -G pool/token '<<<' "password"

tests:assert-stdout \
    'Hash table pool/token with 100 items successfully created'

tests:assert-test -f $(tests:get-tmp-dir)/tables/pool/token

tests:ensure wc -l '<' $(tests:get-tmp-dir)/tables/pool/token
tests:assert-stdout-re '^100$'
