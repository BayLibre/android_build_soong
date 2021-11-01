#!/bin/bash -eu

# This is a helper script for the update-api family of build targets
# This script will generate a script for updating the source tree

# This indirection is necessary since the source tree will be made read-only
# during builds

function parse_args(){
  options=$(getopt -l "current_api_gen:,current_api_src:,removed_api_gen:,removed_api_src:,out:" -o "" -- "$@")

  eval set -- "$options"
  while true; do
    case $1 in
      --current_api_gen)
        shift;
        current_api_gen=$1
        ;;
      --current_api_src)
        shift;
        current_api_src=$1
        ;;
      --removed_api_gen)
        shift;
        removed_api_gen=$1
        ;;
      --removed_api_src)
        shift;
        removed_api_src=$1
        ;;
      --out)
        shift;
        out=$1
        ;;
      --)
        shift;
        break;
    esac
    shift
  done
}
parse_args $@

# Create file
cat <<EOF > ${out}
#!/bin/bash -eu
cp -f ${current_api_gen} ${current_api_src}
cp -f ${removed_api_gen} ${removed_api_src}
EOF


# Make file executable
chmod +x ${out}
