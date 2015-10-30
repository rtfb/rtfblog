#!/bin/bash

if [[ `service postgresql status | grep -w active` ]]; then
	echo "System-wide PGSQL server is running, refusing to start."
	echo "Run 'sudo service postgresql stop' and try again".
	exit
fi

wait_for_line () {
	while read line
	do
		echo "$line" | grep -q "$1" && break
	done < "$2"
	# Read the fifo for ever otherwise process would block
	cat "$2" >/dev/null &
}

# Start PostgreSQL process for tests
PGSQL_DATA=`mktemp -d /tmp/PGSQL-XXXXX`
PGSQL_PATH=`pg_config --bindir`
${PGSQL_PATH}/initdb ${PGSQL_DATA}
mkfifo ${PGSQL_DATA}/out
${PGSQL_PATH}/postgres -F -k ${PGSQL_DATA} -D ${PGSQL_DATA} &> ${PGSQL_DATA}/out &

# Wait for PostgreSQL to start listening to connections
wait_for_line "database system is ready to accept connections" ${PGSQL_DATA}/out
export RTFBLOG_DB_DRIVER=postgres
export RTFBLOG_DB_TEST_URL="host=${PGSQL_DATA} dbname=template1 sslmode=disable"

goose -path=db/pg/ -env=development up
#${PGSQL_PATH}/psql "$RTFBLOG_DB_TEST_URL" < testdata/testdb.sql

echo "Running tests..."
go test -covermode=count -coverprofile=profile.cov -v ./src/...
exit_status=$?

killall postgres
sleep 0.2   # give some time to remove locks, otherwise rm will fail
rm -r ${PGSQL_DATA}

exit $exit_status
