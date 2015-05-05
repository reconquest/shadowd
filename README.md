# shadowd

**shadowd** it is secure login distribution service, which have two pieces,
server and client.

In a typical server configuration case, you should manually write /etc/shadow
and deploy it on all servers, and all servers will have one same hash in
/etc/shadow, **shadowd** summoned to solve this obscure problem.

For communication between server and client used REST API.

![Plan](https://cloud.githubusercontent.com/assets/8445924/7471437/7d07ceee-f316-11e4-86ed-2e2a06ccf006.png)

## shadowd server configuration

### hash-table

For generating hash table you should run:
```
shadowd -G <login> <password>
```
in this time **shadowd** started generating hash-table, by default **shadowd**
generate table with 2048 hashes of your password with random salt, if you
want, you can change this through flag `-n <amount>` and set specified length of
hash-table, for generating **shadowd** uses `sha256` algorithm, if you want to
use `sha512` you should specify flag `-a sha512`.

Possible options:
- `-t <table_dir>` - use specified directory for storing and reading
    hash-tables. (default: /var/shadowd/ht/)

### SSL certificates

Of course, in client-server communication should be used SSL, and next step of
our trip it's SSL certificates generation.

For generating SSL certificates you should have verified host (**shadowd** server
dnsname) or verified ip address, if you will use current machine for **shadowd**
server, you can skip this step and **shadowd** automatic specify current
hostname and ip address as verified, for setting specified host and ip (may be
more than one) as verified should use `-h <host>` and `-i <address>` arguments.

Possible options:
- `-d <till>` - set time certicicate valid till (default: current
    date plus one year).
- `-b <bytes>` - set specified length of rsa key. (default: 2048)
- `-c <cert_dir>` - use specified directory for storing and reading
    certificates.

And for all this you should run one command:
```
shadowd -C [-h <host>...] [-i <address>...] [another options]
```

After this, `cert.pem`, `private.key` and `public.key` will be saved in specially
directory, by default it's `/var/shadowd/cert/`, which location can be changed
through flag `-c <cert_dir>`.

And finally what you should do it's start **shadowd** server, as mentioned earlier,
**shadowd** is used REST API, by default listening address is `:8080`, so you can
set specified address and port through adding argument `-l <listen>`.

So for running **shadowd** server you should run command:

```
shadowd [-l <listen>]
```

Success, you have configured server, but you need to configure client, for this
you should see
[documentation here](https://github.com/reconquest/shadowc/README.md).
