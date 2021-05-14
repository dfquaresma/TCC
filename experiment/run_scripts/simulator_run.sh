#!/bin/bash
date
set -x

echo "SIM_DURATION: ${SIM_DURATION:=70m}"
echo "LAMBDA: ${LAMBDA:=20}"

echo "OUTPUT_PATH: ${OUTPUT_PATH:=/home/david/TCC/TCC/results/simulation/}"
echo "WARMUP: ${WARMUP:=0}"
echo "SCHEDULER: ${SCHEDULER:=0}"
echo "ID: ${ID:=00}"
echo "NUMBER_OF_INPUTS: ${NUMBER_OF_INPUTS:=32}"
echo "INPUT_PATH: ${INPUT_PATH:=/home/david/TCC/TCC/results/measurements/}"


inputs="${INPUT_PATH}input-lambda0-1.csv"
for id in `seq 2 ${NUMBER_OF_INPUTS}`;
do
    inputs="${inputs},${INPUT_PATH}input-lambda0-${id}.csv"
done

../alter-faas-simulator/serverless --duration=${SIM_DURATION} --lambda=${LAMBDA} --output=${OUTPUT_PATH} --warmup=${WARMUP} --scheduler=${SCHEDULER} --scenario="lambda${LAMBDA}-idleness300s-warmup${WARMUP}-id${ID}" --inputs=${inputs}
