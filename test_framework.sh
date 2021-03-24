REAL_TOP="$(readlink -f "$(dirname "$0")"/../..)"

TESTS=()

for arg in "$@"; do
  TESTS+=($arg)
done

function create_mock_top() {
  if [[ ! -z "$HARDWIRED_MOCK_TOP" ]]; then
    MOCK_TOP="$HARDWIRED_MOCK_TOP"
    rm -fr "$MOCK_TOP"
    mkdir -p "$MOCK_TOP"
  else
    MOCK_TOP=$(mktemp -t -d st.XXXXXX)
    trap 'cd / && rm -fr "$MOCK_TOP"' EXIT
  fi

  echo "Mock top path: $MOCK_TOP"
  cd "$MOCK_TOP"
}

function fail {
  echo ERROR: $1
  exit 1
}

function copy_directory() {
  local dir="$1"
  local parent="$(dirname "$dir")"

  mkdir -p "$MOCK_TOP/$parent"
  cp -R "$REAL_TOP/$dir" "$MOCK_TOP/$parent"
}

function symlink_file() {
  local file="$1"

  mkdir -p "$MOCK_TOP/$(dirname "$file")"
  ln -s "$REAL_TOP/$file" "$MOCK_TOP/$file"
}

function symlink_directory() {
  local dir="$1"

  mkdir -p "$MOCK_TOP/$dir"
  # We need to symlink the contents of the directory individually instead of
  # using one symlink for the whole directory because finder.go doesn't follow
  # symlinks when looking for Android.bp files
  for i in $(ls "$REAL_TOP/$dir"); do
    local target="$MOCK_TOP/$dir/$i"
    local source="$REAL_TOP/$dir/$i"

    if [[ -e "$target" ]]; then
      if [[ ! -d "$source" || ! -d "$target" ]]; then
        fail "Trying to symlink $dir twice"
      fi
    else
      ln -s "$REAL_TOP/$dir/$i" "$MOCK_TOP/$dir/$i";
    fi
  done
}

# This test runner is wrong in a hundred ways:
# - The test sharding protocol is not supported
# - --test_filter probably doesn't work
# - Test cases that were not run are not mentioned in test.xml
# - We don't care about quoting test names in text.xml
# - Failure messages are not mentioned to test.xml
# - test.xml doesn't have a test class name
# - The semantics of test.xml are probably wrong in many other ways
# - It's probably very easy for a test case to confuse the test runner if
#   it exits in an unexpected way
# - Qu

function run_test_suite() {
  if [[ ${#TESTS[@]} -eq 0 ]]; then
    TESTS=()
    for i in $(declare -F | cut -d' ' -f3 | grep "^test_"); do
      TESTS+=($i)
    done
  fi

  local test_name
  local test_results=()

  for test_name in ${TESTS[*]}; do
    echo "---------------------------------- $test_name"
    local test_result=FAILURE
    (
      eval "$test_name"
    ) && test_result=SUCCESS

    if [[ "$test_result" == "SUCCESS" ]]; then
      echo "$test_name: SUCCESS"
    else
      echo "$test_name: FAIL"
    fi

    test_results+=("$test_result")
  done

  if [[ ! -v XML_OUTPUT_FILE ]]; then
     return
  fi

  echo "<?xml version='1.0' encoding='UTF-8'?>" >> "$XML_OUTPUT_FILE"
  echo "<testsuites name='shell test'>" >> "$XML_OUTPUT_FILE"
  echo "  <testsuite name='shell test'>" >> "$XML_OUTPUT_FILE"

  local failure_seen=

  for i in ${!TESTS[@]}; do
    local test_name="${TESTS[$i]}"
    local test_result="${test_results[$i]}"

    echo "    <testcase name='$test_name' status='run'>" >> "$XML_OUTPUT_FILE"
    if [[ "$test_result" == "FAILURE" ]]; then
      echo "      <failure/>" >> "$XML_OUTPUT_FILE"
      failure_seen=true
    fi
    echo "    </testcase>" >> "$XML_OUTPUT_FILE"
  done

  echo "  </testsuite>" >> "$XML_OUTPUT_FILE"
  echo "</testsuites>" >> "$XML_OUTPUT_FILE"

  if [[ "$failure_seen" == "true" ]]; then
    exit 1
  fi
}

