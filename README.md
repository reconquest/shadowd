# shadowd

**shadowd** it is the secure login distribution service, which consists of two
parts, server and client.

In a typical server configuration case, you should manually update the
`/etc/shadow` and copy it on all servers, and every servers will have same hash in
the `/etc/shadow`, afterwards all servers will have same password hashes in the
`/etc/shadow`. Supposed that attacker successfully gained access to one of your
servers and found collision to that single hash, so afterwards *attacker
actually got access to all servers with that hash*.

**shadowd solution** is to generate hash tables of specified passwords mixed
with random salt for specified users and guarantee that a client receive unique
hash.

If attacker gained user access to server and try to repeat request to
shadowd server and get actual hash during the hash TTL period (one day by
default), then shadowd will give him another unique hash entry. Actually,
**shadowd** can give only two unique hash entries for hash TTL period for one
client, first hash entry may be received only for first request per hash TTL
period, in all later requests will received another hash.

**shadowd** is summoned to solve that obscure problems.

REST API is used for communication between server and client.

![Plan](https://cloud.githubusercontent.com/assets/8445924/7489851/95b5c748-f3ca-11e4-9487-bc4daeedc385.png)

## shadowd configuration

1. [Generate hash tables](#hash-tables)
2. [Generate SSL certificates](#ssl-certificates)
3. [Start shadowd](#start-shadowd)

### Hash Tables

For generating hash table you should run:
```
shadowd [options] -G <login> <password>
```
**shadowd** will generate hash table with 2048 hashed entries of specified
password, hash table size can be specified via flag
`-n <amount>` `sha256` will be used as default hashing algorithm, but `sha512`
can be used via `-a sha512` flag.

### SSL certificates

Assume attacker gained access to your servers, he can wait for next
password update and do man-in-the-middle attack, afterwards passwords on
servers will be changed on him password and get more access to the servers.

For solving that problem one should use SSL certificates, which confirms
authority of the login distribution server.

For generating SSL certificates you should have trusted host (shadowd server
DNS name) or trusted ip address, if you will use localhost for shadowd
server, you can skip this step and shadowd will automatically specify current
hostname and ip address as trusted, in other cases you should pass options for
setting trusted hosts/addresses of shadowd.

Possible Options:
- `-h <host>` - set specified host as trusted. (default: current hostname)
- `-i <address>` - sett specified ip address as trusted. (default: current ip
    address)
- `-d <till>` - set time certicicate valid till (default: current
    date plus one year).
- `-b <bytes>` - set specified length of RSA key. (default: 2048)

And for all of this you should run one command:
```
shadowd [options] -C [-h <host>...] [-i <address>...]
```

Afterwards, `cert.pem` and `key.pem` will be stored in
`/var/shadowd/cert/` directory, which location can be changed through flag
`-c <cert_dir>`.

### Start shadowd

As mentioned earlier, shadowd uses REST API, by default listening on `:8080`,
but you can set specified address and port through passing argument
`-L <listen>`:

```
shadowd [options] [-L <listen>] [-a <hash_ttl>]
```

For setting hash TTL duration you should pass `-a <hash_ttl>` argument, by
default hash TTL is `24h`.

#### General options:

- `-c <cert_dir>` - use specified directory for storing and reading
    certificates.
- `-t <table_dir>` - use specified directory for storing and reading
    hash tables. (default: /var/shadowd/ht/)


Success, you have configured server, but you need to configure client, for this
you should see
[documentation here](https://github.com/reconquest/shadowc/README.md).
