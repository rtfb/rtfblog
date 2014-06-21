#!/bin/sh

echo "package main" > src/version.go
echo -n "const generatedVersionString = " >> src/version.go
echo "\"Dev build @ <`git rev-parse --short HEAD`>\"" >> src/version.go
