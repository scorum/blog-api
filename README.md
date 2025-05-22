# Blog API
A RESTful API service for a blog platform built with Go. This API provides endpoints for managing blog posts, user authentication, and database operations using PostgreSQL with PostGIS extension.


## Dependency
dep is used for dependency management
```
dep ensure --vendor-only
```

## Build verify
```
cd verify/c-secp256k1
./autogen.sh && ./configure --enable-module-recovery  && make && make install
```
### Also for linux install libbssl-dev and libgmp (for ubuntu, mint)
```
apt-get install libssl1.0-dev, libgmp
```

## Database
## To start DB container
```docker run -d -p 5432:5432 mdillon/postgis:9.6-alpine```  

### Migration
SQL migrate is used https://github.com/rubenv/sql-migrate.
```
go get -v github.com/rubenv/sql-migrate/...
```
sql-migrate up