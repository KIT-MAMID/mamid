

  ```
  # Setup the $GOPATH environment variable
  # Checkout this repository to $GOPATH/src/github.com/KIT-MAMID/mamid
  cd $GOPATH/src/github.com/KIT-MAMID/mamid
  git submodule update --fetch
  make
  ```

* Running tests requires a PostgreSQL instance with permission to `CREATE DATABASE` and `DESTROY DATABASE`
  * Running the test instance in a docker container different than the one used for `make testbed_*` builds is recommended

    ```
    docker run --name mamid-postgres-tests -e POSTGRES_PASSWORD=mysecretpassword -d postgres
    # You should now be able to connect to the container using the password above
    psql postgres
    # You can run tests by setting the appropriate DSN environment variable
    MAMID_TESTDB_DSN="host=localhost sslmode=disable dbname=postgres" make test-verbose
    ```

* Creating the msp certificate infrastructure
  * Generate a ca by running `./scripts/generateCA.sh`
     * The ca certificate will be located in `./ca/mamid.pem`
     * The ca's private key will be located in `./ca/private/mamid_private.pem`
  * Generate slave certs & keys by running `./scripts/generateAndSignSlaveCert.sh <hostname>`
     * If you can't use hostnames, you can add an IP to the cert as well by running `./scripts/generateAndSignSlaveCert.sh <hostname> <IP>`
     * Slave certs & keys are located in `./ca/slaves/<slavename>/`

### Master

### Slave

### Notifier



