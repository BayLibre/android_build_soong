parseReadelfOutput() {
  while IFS= read -r line
  do
      if [[ $line = *FUNC*GLOBAL*UND*@* ]] ;
      then
          echo "$line" | sed -r 's/.*UND (.*)@.*/\1/g' >> "$2"
      fi
  done < "$1"
  echo "" >> "$2"
}

unzipJarAndApk() {
  tmpUnzippedDir="$1"/tmpUnzipped
  [[ -e "$tmpUnzippedDir" ]] && rm -rf "$tmpUnzippedDir"
  mkdir "$tmpUnzippedDir"
#  find .. -name "*.jar" -exec unzip -d "$tmpUnzippedDir" \;
#  find .. -name "*.apk" -exec unzip -d "$tmpUnzippedDir" \;
#  find . -name "*.MF" -exec rm {} \;
}

lookForExecFile() {
  dir="$1"
  readelf="$2"
#  find "$dir" -name "*.so" -exec "$2" --dyn-symbols {} > "$dir"/../tmpReadelf.txt \;
  find "$dir" -type f -perm /111 -exec "$2" --dyn-symbols {} >> "$dir"/../tmpReadelf.txt \;
}

unzipJarAndApk "$2"
lookForExecFile "$2" "$3"
tmpReadelfOutput="$2/../tmpReadelf.txt"
[[ -e "$4" ]] && rm "$4"
parseReadelfOutput "$tmpReadelfOutput" "$4"
#[[ -e "$tmpReadelfOutput" ]] && rm "$tmpReadelfOutput"
rm -rf "$2/tmpUnzipped"

