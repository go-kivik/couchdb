#!/bin/bash -e

curl --silent --fail -o /dev/null -X PUT ${1}_config/replicator/connection_timeout -d '"5000"'
