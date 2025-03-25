#!/bin/bash

# Run the first command
python3 crawl.py
if [ $? -ne 0 ]; then
  echo "crawl.py failed"
  exit 1
fi

# Run the second command
./wait-for-data.sh
if [ $? -ne 0 ]; then
  echo "cronjob failed"
  exit 1
fi