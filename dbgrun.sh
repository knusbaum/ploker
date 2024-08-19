#!/usr/bin/env bash

make
pushd out
./server
popd
