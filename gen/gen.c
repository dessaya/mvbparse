// gen.c: lee de un CSV creado a partir de mvb_signal.py, y escupe la secuencia de 1s y 0s para regenerar la señal MVB.

#include <stdio.h>
#include <time.h>
#include <string.h>
#include <stdlib.h>
#include <stdint.h>

#define BT 666.66666666666666666e-9

void sleep(double seconds) {
    printf("sleep for %f nanoseconds\n", seconds * 1000000000);
}

void sleep_until(double until) {
    printf("sleep until t = %f\n", until);
}

void sleep_random_master_slave() {
    // sleep between 4 - 16 microseconds
    double amount = rand() % 12 + 4;
    sleep(amount / 1000000);
}

typedef enum { BIT_0, BIT_1, BIT_NH, BIT_NL, } bit_t;

void send_high() {
    printf("HIGH\n");
    sleep(BT / 2);
}

void send_low() {
    printf("LOW\n");
    sleep(BT / 2);
}

// 3.3.1.2 Bit encoding
void send_bit(bit_t bit) {
    switch (bit) {
        case BIT_1:
            send_high();
            send_low();
            break;
        case BIT_0:
            send_low();
            send_high();
            break;
        case BIT_NH:
            send_high();
            send_high();
            break;
        case BIT_NL:
            send_low();
            send_low();
            break;
    }
}

// 3.3.1.4 Start Bit
void send_start_bit() {
    send_bit(BIT_1);
}

// 3.3.1.5 Start Delimiter
void send_master_start_delimiter() {
    send_bit(BIT_NH);
    send_bit(BIT_NL);
    send_bit(BIT_0);
    send_bit(BIT_NH);
    send_bit(BIT_NL);
    send_bit(BIT_0);
    send_bit(BIT_0);
    send_bit(BIT_0);
}

// 3.3.1.5 Start Delimiter
void send_slave_start_delimiter() {
    send_bit(BIT_1);
    send_bit(BIT_1);
    send_bit(BIT_1);
    send_bit(BIT_NL);
    send_bit(BIT_NH);
    send_bit(BIT_1);
    send_bit(BIT_NL);
    send_bit(BIT_NH);
}

// 3.3.1.6 End Delimiter
void send_end_delimiter() {
    send_bit(BIT_NL);
    send_bit(BIT_NH);
}

void send_byte(uint8_t byte) {
    for (int i = 7; i >= 0; i--) {
        bit_t bit = (byte >> i) & 0x1;
        send_bit(bit);
    }
}

// send up to 64 bits, return pointer to unsent data
const char *send_bytes(const char *hex) {
    int amount = 0;
    while (hex[0] && hex[1]) {
        char byte_hex[3] = {hex[0], hex[1], '\0'};
        uint8_t byte = strtol(byte_hex, NULL, 16) & 0xff;
        send_byte(byte);
        hex += 2;
        amount++;
        if (amount == 8) {
            break;
        }
    }
    return hex;
}

// TODO 3.4.1.3 Check Sequence
void send_check_sequence() {
    send_byte(0xaa);
}

// 3.4.1.1 Master Frame format
void send_master(const char *hex) {
    send_start_bit();
    send_master_start_delimiter();
    send_bytes(hex);
    send_check_sequence();
    send_end_delimiter();
}

// 3.4.1.2 Slave Frame format
void send_slave(const char *hex) {
    send_start_bit();
    send_slave_start_delimiter();
    do {
        hex = send_bytes(hex);
        send_check_sequence();
    } while (hex[0] && hex[1]);
    send_end_delimiter();
}

int main(void) {
    while (1) {
        char buf[256];
        if (!fgets(buf, 256, stdin))
            break;
        char *ts = strtok(buf, ",");
        double t = strtod(ts, NULL);
        sleep_until(t);
        char *master_hex = strtok(NULL, ",");
        send_master(master_hex);
        char *slave_hex = strtok(NULL, ",");
        sleep_random_master_slave();
        send_slave(slave_hex);
    }
}
