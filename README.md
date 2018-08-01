# parsetx-dextop-2
Script to parse the transaction of dextop version 2.


# How to build
Assuming you have configured your Gopath and setup your Go environment.

Option 1: Without Gopkg

$go get github.com/ethereum/go-ethereum 
$go build

Option 2: With Gopkg

$dep init
$dep ensure
$go build

For option 2, if you run into issue complaining
"fatal error: 'libsecp256k1/include/secp256k1.h' file not found", that is probably due to "dep",
and a temporary solution is to manually copy the files to vendor by doing:
"cp -r ${GOPATH}/src/github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1 vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/".

# How to run
e.g., $./parsetx-dextop-2 0x96f7d81da5d0ecfd9e26e04937a6fa15f31f39f84acb6f9c50a72fad63689857
