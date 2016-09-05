# Installation

## Sample scenario

For the instructions in this manual, the following cluster layout is assumed:
![Possible cluster layout for a single application built on MongoDB](https://cdn.rawgit.com/KIT-MAMID/mamid/doc/doc/cluster_layout.svg)

## Setting up the PKI

### Generating the CA

1. In a secure place, generate a folder for the internal pki of your cluster.
2. Generate a folder for the pki managements scripts and config files, e.g. `mkdir scripts`
3. Download the contents of [the 'scripts' directory](https://github.com/KIT-MAMID/mamid/tree/master/scripts) into `scripts`
4. Run `./scripts/generateCA.sh` to generate the CA.
5. In the wizard, enter any name as common name and enter a secure password for the CA.
6. Enter the password again to let the CA self-sign itself.

The CA certificate of your CA will be in `ca/mamid.pem` and its private key in `ca/private/mamid_private.pem`. 
Keep this file secure. In case of a compromisation of the key, the CA and all signed certificates should be imidietly 
removed and replaced by new ones signed by a new CA. Otherwise the cluster may be hijacked.

### Generating the certificates for the slaves

Assume the slave's _hostname_ at PSU1 is `slave01`.
To generate its certificate and key, run `./scripts/generateAndSignSlaveCert.sh slave01`.
You will be promted to enter the CA's password and confirm the signing process.
After the successful signature the slave's cert and key are located in `./ca/slaves/slave01/`.

Repeat this procedure with all other slaves in your cluster.

#### Generating certificates for slaves without a hostname

If you don't have slaves (from master's view) identified with a hostname, you can also sign a ip:
`./scripts/generateAndSignSlaveCert.sh slave01 <IP>`

## Deploying MAMID

### Master

The master can run in different modes. This manual assumes the most secure operation mode.

1. Install the postgresql and make it acessible for a user.
2. Create a database for the configuration. It will be used as configuration store of mamid and can be backuped with 
tools like `pg_dump`,
2. Deploy the master binary on the master.
3. Generate and sign a key/cert pair using the above method for the master.
As name `master` can safely choosen since the common name of this certificate doesn't matter in the verification process.
4. Deploy a copy of the CA certificate (`mamid.pem`), the master key and the master certificate on the master.
5. Deploy a X.509 certificate and key for the web interface (may be signed by your internal CA or an external Authority)
6. Start the master:

        /path/to/your/master \
               -db.dsn "host=localhost port=5433 sslmode=disable dbname=<database> user=<user> password=<password>" \
               -slave.auth.key "/path/to/your/master/key" \
               -slave.auth.cert "/path/to/your/master/certificate" \
               -slave.verifyCA "/path/to/your/mamid.pem " \
               -api.key "/path/to/your/web/key" \ # omit this line to disable https for the web interface
               -api.cert" /path/to/your/web/certificate" \ # omit this line to disable https for the web interface
               -api.verifyCA "/path/to/your/mamid.pem" \ # omit this line to disable client cert auth

Using this configuration the master will serve the web interface via `https` and requires 
user authentication with a client certificate. Such a certificate can also be created with the method described above
and easily imported to the administrator's browser with the PKCS12 file (.p12) also located
in the `./ca/slaves/<certname>` folder. It is possible to use different CAs for slave authentication,
user authentication and the web interface. It's strongly recommended to use the internal CA for slave
authentication.

For more information about the specific master command line options see `master --help`.

### Slaves

1. Install MongoDB on the slave.
2. Deploy the slave certificate and key on each slave as well as the `mamid.pem`.
3. Deploy the slave software on the slaves.
4. Run the slaves:

        /path/to/your/slave \
                -slave.auth.cert "/path/to/the/slave/cert" \
                -slave.auth.key "/path/to/the/slave/key" \ 
                -data "/path/to/your/mongod/data/root/directory"

For more information about the specific slave command line options see `slave --help`.

### Notifier

1. Deploy the notifier software on a server that is able to reach master's web interface
2. Configure the notifier with its config file. See [the sample file](https://github.com/KIT-MAMID/mamid/blob/master/notifier/config.ini.sample).
E-mails are delivered using a smarthost.
3. Configure contacts in the configured contacts file in the following format for e-mail notifications:

        [eve]
        email=eve@name.tld
4. Launch the notifier:

        /path/to/your/notifier <config.ini>
