solc --abi --overwrite *.sol -o build 
solc --bin --overwrite *.sol -o build
abigen --abi=./build/ERC20Interface.abi --pkg=erc20 --out=ERC20Interface.go
# for filename in ./build/*.abi; do
#      BASE=$(basename $filename)
#      echo $BASE
#      GOFILE=${BASE%.abi}
#      echo $GOFILE
#      abigen --abi=$filename --pkg=erc20 --out=../$GOFILE.go
# done