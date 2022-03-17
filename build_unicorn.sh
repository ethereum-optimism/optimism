#!/bin/bash
if [[ ! -d unicorn2 ]]; then
    git clone https://github.com/geohot/unicorn.git -b dev unicorn2
    #git clone https://github.com/unicorn-engine/unicorn.git -b dev unicorn2
fi

cd unicorn2
cmake . -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Release
#cmake . -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Debug
make -j8

# export LIBUNICORN_PATH for Github CI
# TODO: is this actually needed?
if [[ ! -z "$GITHUB_ENV" ]]; then
    echo "LIBUNICORN_PATH=$(pwd)/unicorn2/" >> $GITHUB_ENV
fi
