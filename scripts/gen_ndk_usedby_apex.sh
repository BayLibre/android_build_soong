parseReadelfOutput() {
  while IFS= read -r line
  do
      if [[ $line = *FUNC*GLOBAL*UND*@* ]] ;
      then
          echo "$line" | sed -r 's/.*UND (.*)@.*/\1/g' >> "$2"
      fi
  done < "$1"
}

unzipJarAndApk() {
  dir="$1"
  tmpUnzippedDir = "$dir"/../tmpUnzipped
  [[ -e "$tmpUnzippedDir" ]] && rm "$tmpUnzippedDir"
  mkdir tmpUnzippedDir
  prev_pwd = "$PWD"
  cd "$dir"/../tmpUnzipped
  find "$dir" -name "*.jar" -exec unzip -o {} \;
  find "$dir" -name "*.apk" -exec unzip -o {} \;
  find "$dir"/../tmpUnzipped -name "*.MF" -exec rm {} \;
  cd "$prev_pwd"
}

lookForExecFile() {
  dir="$1"
  readelf="$2"
  find "$dir" -name "*.so" -exec "$2" --dyn-symbols {} > "$dir"/../tmpReadelf.txt \;
  find "$dir" -type f -executable -exec "$2" --dyn-symbols {} >> "$dir"/../tmpReadelf.txt \;
}

unzipJarAndApk "$2"
lookForExecFile "$2" "$3"
tmpReadelfOutput = "$2/../tmpReadelf.txt"
[[ -e "$4" ]] && rm "$4"
parseReadelfOutput "$tmpReadelfOutput" "$4"
[[ -e "$tmpReadelfOutput" ]] && rm "$tmpReadelfOutput"
rm -rf "$2/../tmpUnzipped"
