:mongod
:shadowd-mongodb-config

tests:ensure \
    :shadowd --no-confirm --length 100 -G pool/token '<<<' "password"

tests:assert-stdout \
    'Hash table pool/token with 100 items successfully created'

tests:ensure :mongo "db.shadows.find({}).count()"
tests:assert-stdout-re '^100$'
