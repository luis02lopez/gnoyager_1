#!/bin/bash

KEYNAME="$(./gno.land/build/gnokey generate)" && ADDRESS="$(./gno.land/build/gnokey add -recover $KEYNAME)" && echo "$ADDRESS=10000000000ugnot # @luis02lopez" >> gno.land/genesis/genesis_balances.txt

exec "$@"
