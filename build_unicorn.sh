#!/bin/bash
git clone https://github.com/geohot/unicorn.git -b dev unicorn2
cd unicorn2
cmake . -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Debug
make -j8

