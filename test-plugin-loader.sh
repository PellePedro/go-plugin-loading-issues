#!/bin/ash
if [ $# -gt 0 ]
  then
    LDPATH=-trimpath
fi

echo  "Building plugin with linking mode [$LDPATH]"

cd plugin
PLUGIN=thrift.so
[ -f ${PLUGIN} ] && rm ${PLUGIN}
echo "Building Plugin"
go mod download
go mod tidy
go build -buildmode=plugin ${LDPATH} -o ${PLUGIN} plugin.go
[ -f ${PLUGIN} ] && mv ${PLUGIN} ..
cd -

cd master
BINARY=server
[ -f ${BINARY} ] && rm ${BINARY}
echo "Building ${BINARY}"
go mod download
go mod tidy
go build -o ${BINARY} ${LDPATH} .
[ -f $BINARY ] && mv ${BINARY} ..
cd -

echo "Running ${BINARY}"
./${BINARY}
