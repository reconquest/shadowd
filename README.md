# shadowd

![shadow horse](https://cloud.githubusercontent.com/assets/8445924/9289438/97f8a2e8-435f-11e5-853c-255a7fe22d08.png)

**shadowd** is the secure login distribution service, which consists of two
parts: server and client.

In a typical server configuration case you should manually update the
`/etc/shadow` and copy it on all servers (or via automatic configuration
system); afterwards each server will have same hash in the `/etc/shadow`.
Supposed that attacker successfully gained access to one of your servers and
found collision to that single hash *attacker actually got access to all
servers with that hash*.

**shadowd** is summoned to solve that obscure problem.

**shadowd solution** is to generate hash tables for specified passwords mixed
with random salt for specified users and guarantee that a client receive unique
hash.

One **shadowd** instance can be used for securely instanciate thousands of
servers with same root password for each one. Without any doubts about possible
break-in.

If attacker has user (non-root) access to the one server and try to repeat
request to shadowd server and get actual hash during the hash TTL period (one
day by default), then shadowd will give him another unique hash entry.
Actually, **shadowd** can give only two unique hash entries for hash TTL period
for one client, first hash entry may be received only for first request per
hash TTL period, all other requests will be served by another hash.

If attacker has root access to the one server and will try to brute-force
hash entry for root from `/etc/shadow`, it will not give him any access for
other servers with same password, because **shadowd** will give different
hashes for each server.

If attacker will gain root access to the **shadowd** server (worst-case
scenario), it's will be very time-consuming to brute-force thousands of hashes
without any knowledge about which server is using specific hash entry.

**shadowd** can act as SSH keys publisher too.

**shadowd** can also configure with passwords containers or any other type of
nodes in your infrastructure.

REST API is used for communication between server and client.

![Plan](https://cloud.githubusercontent.com/assets/8445924/7489851/95b5c748-f3ca-11e4-9487-bc4daeedc385.png)

## shadowd configuration

1. [Generate hash tables](#hash-tables)
2. [Generate SSL certificates](#ssl-certificates)
3. [Start shadowd](#start-shadowd)
4. [Adding SSH keys](#adding-ssh-keys)
5. [REST API](#rest-api)

### Hash Tables

For generating hash table you should run:
```
shadowd [options] -G <token>
```
**shadowd** will prompt for a password for specified user token, and after that
will generate hash table with 2048 hashed entries of specified password, hash
table size can be specified via flag `-n <amount>` `sha256` will be used as
default hashing algorithm, but `sha512` can be used via `-a sha512` flag.

Actually, user token can be same as login, but if you want to use several
passwords for same username on different servers, you should specify `<token>`
as `<pool>/<login>` where `<pool>` it is name of role (`production` or `testing`
for example).

### SSL certificates

Assume that attacker gained access to your servers, then he can wait for next
password update and do man-in-the-middle attack, afterwards passwords on
servers will be changed on his password and he will get more access to the
servers.

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

Since client needs certificate, you should copy `cert.pem` on
server with client to `/etc/shadowc/cert.pem`.

### Start shadowd

As mentioned earlier, shadowd uses REST API, by default listening on `:8080`,
but you can set specified address and port through passing argument
`-L <listen>`:

```
shadowd [options] [-L <listen>] [-a <hash_ttl>]
```

For setting hash TTL duration you should pass `-a <hash_ttl>` argument, by
default hash TTL is `24h`.

TTL is amount of time after which shadowd will serve different unique pair of
hash entries to the same requesting client.

#### General options:

- `-c <cert_dir>` - use specified directory for storing and reading
    certificates.
- `-t <table_dir>` - use specified directory for storing and reading
    hash tables. (default: /var/shadowd/ht/)
- `-k <keys_dir>` - use specified dir for reading ssh-keys.
    (default: /var/shadowd/ssh/).

Success, you have configured server, but you need to configure client, for this
you should see
[documentation here](https://github.com/reconquest/shadowc).

### Adding SSH keys

**shadowd** can act as public ssh keys distribution service. Keys can be added
per token by using command:

```
shadowd -K <token>
```

After that command **shadowd** will wait for public SSH key to be entered on
stdin.  Then, specified key will be added to keys list, which is stored under
the directory, set by `-k` flag (/var/shadowd/ssh/ by default).

Optionally, key file can be truncated by using flag `-r` to the standard `-K`
invocation.

**shadowd** will serve that keys by HTTP, as mentioned in following section.

### REST API

**shadowd** offers following REST API:

* `/t/<token>`, where token can be any string, possibly containing slashes.
  Most common interpretation for `<token>` is `<pool>/<username>`, e.g.
  `dev/v.pupkin`.

  `GET` on this URL will return unique hash for specified `<token>` from
  pre-generated via `shadowd -G` hash-table. The first requested hash
  guaranteed to be unique among different hosts, The second and later `GET`
  requests will return same hash again and again until `<hash_ttl>` expires.
  `<hash_ttl>` is configured on server by `-a` flag. Working in that way
  **shadowd** offers secure way of transmitting one hash only once, and
  legitimate client (e.g. **shadowc**) can always be sure that hash, obtained
  from **shadowd**, has not been transferred to someone else on that host.

* `/ssh/<token>`, where `<token>` is same as above.

  `GET` on this URL will return SSH keys, that has been added by `shadowd -K`
  command in `authorized_keys` format (e.g. key per line).

  No special security restrictions apply on that requests.

* `/v/<token>/<hash>`, where `<token>` is same as above, `<hash>` is
    hash of a password, if `<token>` contains `/` (`<pool>/<username>`) then
    the last part of URI will be used as `<hash>`.
    i.e.
    `dev/v.pupkin/$5$NqjWijImgEOapAJw$NL5K7g8SwMferONwiskz6bcluwlO7zbu3V/ZyLzavZD`

  `GET` on this URL will return HTTP status `200 OK` if specified hash exists
  in hash table for specified token, in other case, `404 Not
  Found` will be returned.

  No special security restrictions apply on that requests.
