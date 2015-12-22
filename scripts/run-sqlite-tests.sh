#!/bin/bash

export RTFBLOG_DB_DRIVER=sqlite3
export RTFBLOG_DB_TEST_URL="test.db"

goose -path=db/sqlite/ -env=development up
#${PGSQL_PATH}/psql "$RTFBLOG_DB_TEST_URL" < testdata/testdb.sql

echo "Running tests on $RTFBLOG_DB_DRIVER..."
go test -covermode=count -coverprofile=profile.cov -v ./src/...
exit_status=$?

rm -r $RTFBLOG_DB_TEST_URL

exit $exit_status
