#ifndef INPUT_H
#define INPUT_H

#include <stdio.h>
#include <stdbool.h>

void input_init(FILE *f, bool inverted);

bool input_skip(double seconds);
bool input_wait_until(bool v);

bool input_get();
double input_t();
size_t input_n();

void input_trace(bool v);

#endif
