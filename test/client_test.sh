#!/bin/bash
SCRIPT_DIR=$(dirname "$(realpath "$0")")
cd "$SCRIPT_DIR" || exit 1

srv_containers=`docker ps --filter=ancestor=adit_srv:latest --format "{{.ID}} {{.Image}} {{.CreatedAt}}" | sort -k3,3`

if [ "$(echo "$srv_containers" | wc -l)" -gt 1 ]; then
    echo "too many adit_srv containers running, using latest"
fi
srv_container_id=`echo $srv_containers | awk 'NR==1 {print $1}'`

test_dir=$(pwd)
cd ../client
go mod download && go test .
if [ $? -ne 0 ]; then
    echo "error running tests"
    exit 1
fi
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$test_dir/adit-client"
if [ $? -ne 0 ]; then
    echo "error building binary"
    exit 1
fi

cd "$test_dir"
chmod +x adit-client
#create 100mb file for test
dd if=/dev/zero of=100mb.file bs=1M count=100

[ -p pipefile ] && rm pipefile
mkfifo pipefile
./adit-client -r "ws://localhost:8080/ws" -i 100mb.file -vvv | tee pipefile &
TIMEOUT_DURATION=3
while IFS= read -r -t $TIMEOUT_DURATION line
do
    if [[ "$line" == *"Phrase generated for file transfer:"* ]]; then
        phrase=$(echo "$line" | awk -F': ' '{print $2}')
        ./adit-client -r "ws://localhost:8080/ws" -c $phrase -vvv
    fi
done < pipefile

if [ $? -eq 142 ]; then #timeout
    echo "error failed integration test"
    exit 1
fi

echo done

    