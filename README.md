# Go database/sql Retry Bug Tester

database/sql will automatically retry queries when the underlying connection is
broken. This test application is an experiment to determine if database/sql's
automatic retry logic is safe.

## Hypothesis

If database/sql executes a non-idempotent query, and that query is interrupted,
the automatic retry logic may cause the query to execute multiple times.

## Experiment

This application will connect to a database over a purposefully unreliable
connection by using [cavein](https://github.com/jackc/cavein).

This application creates a simple table with one row with a value of 0.

    create table t(n int not null);
    insert into t(n) values(0);

It then executes 1,000 update statements that increment that row.

    update t set n=n+1;

Finally, it selects that value. If there is a discrepancy between that value and
the number of queries that database/sql reported were successful, then it
appears that the retry logic is incorrect.

It connects to PostgreSQL with both the [pq](https://github.com/lib/pq) and the
[pgx](https://github.com/jackc/pgx) drivers to ensure results are not specific
to one database driver.

## Installation

    go get -u github.com/jackc/go_database_sql_retry_bug

## Usage

Create a database for the test.

    createdb go_database_sql_retry_bug

Start the [cavein](https://github.com/jackc/cavein) tunnel proxy that will
introduce connection drops.

    cavein -local=localhost:2999 -remote=localhost:5432 -minbytes=10000 -maxbytes=20000

In another terminal run the test application (supply any needed standard PG*
environment variables).

    PGPORT=2999 PGPASSWORD=secret go_database_sql_retry_bug

## Results

With further information from https://github.com/golang/go/issues/11978 it appears that this is not a bug in database/sql, but in the database driver(s).

    Testing with: github.com/jackc/pgx/stdlib

    Setup database:
    drop table if exists t;
    create table t(n int not null);
    insert into t(n) values(0);

    Exec `update t set n=n+1` 1000 times
    Reported errors: 6
    Reported successes: 994
    Actual value of `select n from t`: 1000

    Testing with: github.com/lib/pq

    Setup database:
    drop table if exists t;
    create table t(n int not null);
    insert into t(n) values(0);

    Exec `update t set n=n+1` 1000 times
    Reported errors: 0
    Reported successes: 1000
    Actual value of `select n from t`: 1001

In both cases, the actual final value is higher than the number of reported
successes. This is to be expected. But in the latter case with pq, the final
value is actually higher than the total number of attempts. This appears to be
a bug with pq.

To reproduce results, run cavein with the following arguments:

    cavein -local=localhost:2999 -remote=localhost:5432 -minbytes=10000 -maxbytes=20000 -seed=5577006791947779410
