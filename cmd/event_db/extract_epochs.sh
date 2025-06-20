#!/bin/bash

if [ "$#" -ne 4 ]; then
    echo "Usage: $0 <input_db> <output_db> <from_epoch> <to_epoch>"
    exit 1
fi

INPUT_DB=$1
OUTPUT_DB=$2
FROM_EPOCH=$3
TO_EPOCH=$4

SQL_SCRIPT=$(cat <<EOF
ATTACH DATABASE '$INPUT_DB' AS source;
BEGIN TRANSACTION;

CREATE TABLE Event AS 
SELECT * FROM source.Event WHERE $FROM_EPOCH <= EpochId AND EpochId <= $TO_EPOCH;

CREATE TABLE Cheater AS
SELECT * FROM source.Cheater WHERE $FROM_EPOCH <= EpochId  AND EpochId <= $TO_EPOCH;

CREATE TABLE Validator AS
SELECT * FROM source.Validator WHERE $FROM_EPOCH <= EpochId AND EpochId <= $TO_EPOCH;

CREATE TABLE BlockEvent AS
SELECT * FROM source.BlockEvent WHERE EventId IN (SELECT EventId FROM Event);

CREATE TABLE Block AS
SELECT * FROM source.Block WHERE AtroposId IN (SELECT EventId FROM Event);

CREATE TABLE Atropos AS
SELECT * FROM source.Atropos WHERE AtroposId IN (SELECT EventId FROM Event);

CREATE TABLE Tx AS
SELECT * FROM source.Tx WHERE EventId IN (SELECT EventId FROM Event);

CREATE TABLE Parent AS
SELECT * FROM source.Parent WHERE EventId IN (SELECT EventId FROM Event);

COMMIT;

DETACH DATABASE source;
EOF
)

sqlite3 "$OUTPUT_DB" "$SQL_SCRIPT"