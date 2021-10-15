#ifndef MBV_GEN_H
#define MBV_GEN_H

#include <stdint.h>
#include <stddef.h>

typedef struct sendbuf {
    uint8_t data[128];
    size_t bytes;
    uint8_t bits;
} sendbuf_t;

sendbuf_t *next_frame();

#endif
