#!/bin/bash
if [[ ! -d unicorn2 ]]; then
    git clone https://github.com/geohot/unicorn.git -b dev unicorn2
    #git clone https://github.com/unicorn-engine/unicorn.git -b dev unicorn2
fi

cd unicorn2
cmake . -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Release
#cmake . -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Debug
make -j8
