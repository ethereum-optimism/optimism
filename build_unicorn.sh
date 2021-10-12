#!/bin/bash
git clone https://github.com/geohot/unicorn.git
cd unicorn
UNICORN_ARCHS=mips make -j8
UNICORN_ARCHS=mips make -j8

