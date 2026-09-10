#!/bin/bash
for i in {1..12}; do
  curl -i http://localhost:8080/data
  echo ""
done
