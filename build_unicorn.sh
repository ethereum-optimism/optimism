#!/bin/bash
if [[ ! -d unicorn2 ]]; then
    git clone https://github.com/geohot/unicorn.git -b dev unicorn2
    #git clone https://github.com/unicorn-engine/unicorn.git -b dev unicorn2
fi

cd unicorn2
cmake . -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Release
#cmake . -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Debug
make -j8

# setting this avoids re-building unicorn in setup.py
export LIBUNICORN_PATH=$(pwd)

cd bindings/python
sudo python3 setup.py install
