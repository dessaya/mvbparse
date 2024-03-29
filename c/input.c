#include "input.h"

#include <stdint.h>

// samples per second
#define SAMPLE_RATE 12000000

#define SIGNAL_HIGH 0x02
#define SIGNAL_LOW 0x00

struct input {
    bool done;
    bool current;
    size_t n;
    bool trace;
};

static struct input input;

void input_init() {
    freopen(NULL, "rb", stdin);
    input.done = false;
    input.current = false;
    input.n = 0;
    input.trace = false;
}

static bool input_next_sample() {
    if (input.done) {
        return false;
    }
    uint8_t b;
    size_t n = fread(&b, 1, 1, stdin);
    if (n == 0) {
        input.done = true;
        return false;
    }
    input.current = b == SIGNAL_HIGH;
    input.n++;
    if (input.trace) {
        printf("%zd %d\n", input.n, input.current);
    }
    return true;
}

bool input_skip(double seconds) {
    size_t n_samples = seconds * SAMPLE_RATE;
    for (size_t i = 0; i < n_samples; i++) {
        bool r = input_next_sample();
        if (!r) return false;
    }
    return true;
}

bool input_wait_until(bool v) {
    while (input.current != v) {
        bool r = input_next_sample();
        if (!r) return false;
    }
    return true;
}

bool input_get() {
    return input.current;
}

double input_t() {
    return input.n / (double)SAMPLE_RATE;
}

size_t input_n() {
    return input.n;
}

void input_trace(bool v) {
    input.trace = v;
}
