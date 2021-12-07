#include <stdio.h>

// From an API surface
#include <log/log_id.h>

int main() {
  printf("Hello Multitree World\n");
  __android_log_write(0, "tag", "log"); // From the API surface
  return 0;
}
