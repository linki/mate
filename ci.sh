TOOLCHAIN=${TOOLCHAIN:-"bash -c"}
APP_NAME=${APP_NAME:-"$(basename $(pwd))"}
TEAM_NAME=${TEAM_NAME:-"teapot"}
GOPATH=${GOPATH:-"/go"}
GODIR="."

setup_godir() {
    GOPATH="$(pwd)/build/go"
    GODIR="$GOPATH/src/github.bus.zalan.do/$TEAM_NAME/$APP_NAME"
    # clean possible previous godir
    rm -f "$GODIR"
    $TOOLCHAIN "mkdir -p $GOPATH/{src,bin}"
    $TOOLCHAIN "mkdir -p $GOPATH/src/github.bus.zalan.do/$TEAM_NAME"
    $TOOLCHAIN "ln -s $(pwd) $GODIR"
}

if [ -n "$BUILD_NUMBER" ]; then
    setup_godir
else
    echo "This script is meant to be run on a CI"
    exit 1
fi

_() {
    $TOOLCHAIN "cd $GODIR && GOPATH=$GOPATH $*"
}

_ make build.push
