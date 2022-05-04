cd unicorn
mkdir -p build
cd build
cmake .. -DUNICORN_ARCH=mips -DCMAKE_BUILD_TYPE=Release
make -j8

# The Go linker / runtime expects these to be there!
cp libunicorn.so.1 ..
cp libunicorn.so.2 ..

# export LIBUNICORN_PATH for Github CI
# TODO: is this actually needed?
if [[ ! -z "$GITHUB_ENV" ]]; then
    echo "LIBUNICORN_PATH=$(pwd)/unicorn/" >> $GITHUB_ENV
fi
