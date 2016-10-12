:shadowd-listen "127.0.0.1:60002"

tests:ensure :shadowd -G --no-confirm --length 100 a/b/c/d '<<<' 'old'

tests:ensure curl -X PUT -k "https://127.0.0.1:60002/t/a/b/c/d"

salts=($(cat $(tests:get-stdout-file)))

payload="password=new"
for salt in "${salts[@]}"; do
    tests:ensure python -c "import crypt; print(crypt.crypt('BAD', '\$salt'))"
    payload="$payload&shadow[]=$(cat $(tests:get-stdout-file))"
done

tests:ensure curl -v -X PUT -d "\$payload" -k "https://127.0.0.1:60002/t/a/b/c/d"
tests:assert-stderr '400 Bad Request'
