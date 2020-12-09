#!/bin/bash
go build .
./gameoflife --role client &
./gameoflife --role broker --workers 127.0.0.1:8110#127.0.0.1:8120#127.0.0.1:8130 &
./gameoflife --role worker --workers 127.0.0.1:8110 &
./gameoflife --role worker --workers 127.0.0.1:8120 &
./gameoflife --role worker --workers 127.0.0.1:8130