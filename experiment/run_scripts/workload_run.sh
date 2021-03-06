#!/bin/bash
date
set -x

echo "TARGET: ${TARGET:=https://5vjlxy2vi2.execute-api.us-east-2.amazonaws.com/thumbnailator/}"
echo "EXPI_ID: ${EXPI_ID:=00}"
echo "RESULTS_PATH: ${RESULTS_PATH:=/home/david/TCC/TCC/results/measurements/}"
echo "NUMBER_OF_REQS: ${NUMBER_OF_REQS:=20000}"
echo "LAMBDA: ${LAMBDA:=200}"
echo "TYPE: ${TYPE:=measurement}"

go run ../workload/main.go --target=${TARGET} --exp_id="${TYPE}-lambda${LAMBDA}-${EXPI_ID}.csv" --results_path=${RESULTS_PATH} --nreqs=${NUMBER_OF_REQS} --lambda=${LAMBDA}
