:mongod
:shadowd-mongodb-config

tests:ensure ssh-keygen -t rsa -b 1024 -f id_rsa
tests:ensure :shadowd -K blah/token '<' id_rsa.pub

tests:ensure ssh-keygen -t rsa -b 1024 -f id_rsa_2
tests:ensure :shadowd -K blah/token '<' id_rsa_2.pub

tests:ensure :mongo "db.keys.find({}, {key:1,_id:0}).pretty()"
tests:assert-no-diff stdout <<KEYS
{
	"key" : "$(cat id_rsa.pub)"
}
{
	"key" : "$(cat id_rsa_2.pub)"
}
KEYS
