# Create Gospace
    cd ~
    mkdir gospace
    cd gospace
    export GOPATH=~/gospace

You might want to put this into your .bashrc

# Get MAMID
    go get github.com/KIT-MAMID/mamid
    go get -u github.com/jteeuwen/go-bindata/...
    cd $GOPATH/src/github.com/KIT-MAMID/mamid
    git submodule update --init

# Unit Tests

Running tests requires a PostgreSQL instance with permission to `CREATE DATABASE` and `DESTROY DATABASE`

 Running the test instance in a docker container different than the one used for `make testbed_*` builds is recommended

    docker run --name mamid-postgres-tests -p 5432:5432 -e POSTGRES_PASSWORD=foo1 -d postgres
    # You should now be able to connect to the container using the password above
    psql -h localhost -U postgres
    # You can run tests by setting the appropriate DSN environment variable
    MAMID_TESTDB_DSN="host=localhost port=5432 user=postgres password=foo1 sslmode=disable dbname=postgres" make test-verbose

# Docker Test Cluster

The Makefile includes targets to create a cluster test environment in docker.

It spawns the master, postgres, the notifier and three slaves.

If you want to spawn more than three slaves you can adjust the `TESTBED_SLAVE_COUNT` variable in the Makefile

## Generate Certificates
For communication between master and slaves certificates are needed for encryption and authentication.
These have to be signed by a local CA.


You can generate a testing CA using

    ./scripts/generateCA.sh

The CA public and private keys will be generated in `ca/mamid.pem` and `ca/private/mamid_private.pem`

You further have to create certificates for the master and the slaves:

    ./scripts/generateAndSignSlave.sh master 10.101.202.1

You can generate multiple certificates for the slaves at once using:

    read -s -p "Enter CA signing key passphrase:: " CA_PASS
    export CA_PASS
    for i in $(seq -f %02g 1 20); do ./scripts/generateAndSignSlaveCert.sh slave${i} 10.101.202.1${i}; done

## Configuring the Notifier

This is optional. If this step is skipped the notifier will exit immediately.

Go into the `notifier` directory and create a config.ini file

    [notifier]
    api_host=http://10.101.202.1:8080
    contacts=contacts.ini
    [smtp]
    relay_host=mail.foo.bar:25 #Your local smarthost
    mail_from=mamid@foo.bar

and a contacts.ini file

    [person-name]
    email=person-email@foo.bar

You should then receive an Email for every problem.

## Starting the Cluster

You can now start the testbed using `make testbed_up`.

This will build MAMID inside a docker container and then start all components.
Containers from previous testbed runs will be deleted.

You should now have a running instance of the master at `10.101.202.1:8080`


# Testing Procedure / Basic Usage

## Adding Slaves

You can now continue by adding and enabling some slaves in the GUI or using the test fixtures script:

    ./scripts/fixtures.py -c -n <number of slaves specified in TESTBED_SLAVE_COUNT>

The slaves should be active and not generate any problems.

## Adding a Replica Set

Continue by adding a Replica Set using the GUI.
After you clicked on "Create" Mongods will be assigned to the Replica Set.

They will be in destroyed state first and a problem might appear, but after about 30 seconds the problem should have disappeared and the Mongods should be in running state. 

## Failure of a Slave

You can kill a slave using `docker stop slave02`. This should generate a problem.

If you then set the killed slave to disabled (and there still is a free slave of the same persistence type left) a new Mongod should be added to the Replica Set.

*Note:* The Mongod on the killed slave will not be removed as MAMID can not communicate with the slave.
For it to be removed you have to either restart the slave or delete the slave using the GUI.

*Note:* For the adding of the new Mongod to work the initial Replica Set has to consist of at least three members. 
Otherwise the Replica Set will not be able to elect a new primary and MAMID can not configure the Replica Set as this can only happen through the primary (without force).

To fix this, the administrator has to log in to the remaining secondary member and force the adding of the new Mongod, e.g.:

    mongo 10.101.202.101:18080
    use admin
    db.auth("mamid", "<password, see system tab>")
    conf = rs.conf()
    conf.members.push({"_id": <id from 0 to 255 not used in config yet>, "host": "<new host:port>"})
    rs.reconfig(conf, {force:true})

## Temporarily Unreachable Slave

To simulate a temporary disconnection of a slave, e.g. by a broken network cable, you can disconnect the container from the network using

    docker network disconnect mamidnet0 slave01

This should generate a problem.

When you reattach the container to the network using

    docker network connect --ip 10.101.202.101 mamidnet0 slave01

the problem should disappear.

## Changing Replica Set Member Counts

If you increase the member counts of a Replica Set new members should appear and be added to the Replica Set.

If not enough free ports are available a problem should appear.

## Destroying a Replica Set

If you click on the *Remove Replica Set* Button in the Replica Set Detail View, the Replica Set should disappear and its Mongods and their data should be destroyed.
