#!/bin/bash

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
#export DB_TEST_URL="postgresql:///?host=${PGSQL_DATA}&dbname=template1"

# Run the tests
#nosetests
