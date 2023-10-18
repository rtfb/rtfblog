#!/bin/bash

export RTFBLOG_DB_DRIVER=sqlite3
export RTFBLOG_DB_TEST_URL="test.db"

$GOPATH/bin/migrate -path=db/sqlite/migrations -database="sqlite3://$RTFBLOG_DB_TEST_URL" up

echo "Running tests on $RTFBLOG_DB_DRIVER..."
go test -covermode=count -coverprofile=profile.cov -v ./src/...
exit_status=$?

rm -r $RTFBLOG_DB_TEST_URL

exit $exit_status
