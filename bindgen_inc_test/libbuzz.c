#include <stdio.h>
#include "libbuzz_generated_header.h"

void fizz(int i, foo* my_foo){
    printf("hello from c! i = %i, my_foo->x = %i\n", i, my_foo->x);
}
