#!/bin/sh

echo "package main"
echo -n "\n"
echo -n "const generatedVersionString = "
echo "\"Dev build @ <`git rev-parse --short HEAD`>\""
