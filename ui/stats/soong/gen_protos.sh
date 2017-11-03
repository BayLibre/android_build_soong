set -e
cd $(dirname $0)
SCRIPT_DIR="$PWD"
cd ../../../../..
CHECKOUT_ROOT="$PWD"
if [ -z "$OUT_DIR" ]; then
  OUT_DIR="out"
fi

BUILD_OUT_DIR="$OUT_DIR"
PROTO_DIR="${SCRIPT_DIR}/proto"
INPUTS="$(ls ${PROTO_DIR}/*.proto)"
SCRIPT_OUT_DIR="${SCRIPT_DIR}/gen"

# because this script removes a source directory, we require the user to pass the '--run' flag
# or else we just print instructions
RUN=false

function usage() {
  echo "$@"
  echo "usage: gen_protos.sh [--run]"
  echo "will remove ${SCRIPT_OUT_DIR} and refill it with generated files"
  exit 1
}

function echoAndDo() {
  echo "$@"
  eval "$@"
}

function generate() {
  echo "Replacing ${SCRIPT_OUT_DIR}"

  #rm -rf "${SCRIPT_OUT_DIR}"
  mkdir -p "${SCRIPT_OUT_DIR}"

  echoAndDo aprotoc --go_out="${SCRIPT_OUT_DIR}" --proto_path="${PROTO_DIR}" "${INPUTS}"

  mv "${SCRIPT_OUT_DIR}"/android/soong/ui/stats/soong/gen/stats.pb.go "${SCRIPT_OUT_DIR}/"
  rm -rf "${SCRIPT_OUT_DIR}/android"

  echo "Done"
}

for arg in "$@"; do
  case "$arg" in
  "--run") RUN=true;;
  *) usage unrecognized argument "$arg";;
  esac
done

if [ "$RUN" == "true" ]; then
  generate
else
  usage
fi
