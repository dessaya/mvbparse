#include <stdint.h>
#include "input.h"

// 3.2.3.1 Signalling speed (bit period in seconds)
#define BT 666.7e-9

#define HIGH true
#define LOW false

// 3.3.1.2 Bit encoding
typedef enum {
    BIT_0, // LOW  HIGH
    BIT_1, // HIGH LOW
    NH,    // HIGH HIGH
    NL,    // LOW  LOW
} symbol_t;

typedef enum {
    AT_NONE,
    AT_LOGICAL,
    AT_DEVICE,
    AT_ALL_DEVICES,
    AT_DEVICE_GROUP,
} address_type_t;

typedef enum {
    MR_PROCESS_DATA,
    MR_RESERVED,
    MR_MASTERSHIP_TRANSFER,
    MR_GENERAL_EVENT, // parameters
    MR_MESSAGE_DATA,
    MR_GROUP_EVENT,
    MR_SINGLE_EVENT,
    MR_DEVICE_STATUS,
} master_request_t;

typedef enum {
    SFS_NONE,
    SFS_SINGLE,
    SFS_PROPOSED_MASTER,
    SFS_DEVICE_GROUP,
    SFS_SUBSCRIBED_SOURCE,
} slave_frame_source_t;

typedef enum {
    SR_NONE,
    SR_PROCESS_DATA,
    SR_MASTERSHIP_TRANSFER,
    SR_EVENT_IDENTIFIER,
    SR_MESSAGE_DATA,
    SR_DEVICE_STATUS,
} slave_response_t;

typedef enum {
    SFD_NONE,
    SFD_SUBSCRIBED_SINKS,
    SFD_MASTER,
    SFD_SELECTED_DEVICES,
    SFD_MASTER_OR_MONITOR,
} slave_frame_destination_t;

typedef struct {
    int n;
    address_type_t address_type;
    master_request_t master_request;
    slave_frame_source_t slave_frame_source;
    int slave_frame_size;
    slave_response_t slave_response;
    slave_frame_destination_t slave_frame_destination;
} fcode_t;

const fcode_t fcodes[] = {
    [ 0 ] = {0, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 16, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
    [ 1 ] = {1, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 32, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
    [ 2 ] = {2, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 64, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
    [ 3 ] = {3, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 128, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
    [ 4 ] = {4, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 256, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
    [ 5 ] = {5, AT_NONE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
    [ 6 ] = {6, AT_NONE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
    [ 7 ] = {7, AT_NONE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
    [ 8 ] = {8, AT_DEVICE, MR_MASTERSHIP_TRANSFER, SFS_PROPOSED_MASTER, 16, SR_MASTERSHIP_TRANSFER, SFD_MASTER},
    [ 9 ] = {9, AT_ALL_DEVICES, MR_GENERAL_EVENT, SFS_DEVICE_GROUP, 16, SR_EVENT_IDENTIFIER, SFD_MASTER},
    [ 10 ] = {10, AT_DEVICE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
    [ 11 ] = {11, AT_DEVICE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
    [ 12 ] = {12, AT_DEVICE, MR_MESSAGE_DATA, SFS_SINGLE, 256, SR_MESSAGE_DATA, SFD_SELECTED_DEVICES},
    [ 13 ] = {13, AT_DEVICE_GROUP, MR_GROUP_EVENT, SFS_DEVICE_GROUP, 16, SR_EVENT_IDENTIFIER, SFD_MASTER},
    [ 14 ] = {14, AT_DEVICE, MR_SINGLE_EVENT, SFS_SINGLE, 16, SR_EVENT_IDENTIFIER, SFD_MASTER},
    [ 15 ] = {15, AT_DEVICE, MR_DEVICE_STATUS, SFS_SINGLE, 16, SR_DEVICE_STATUS, SFD_MASTER_OR_MONITOR},
};

typedef const char *error_t;

const char *EOS = "end of stream";

// 3.3.1.4 Start Bit
error_t wait_until_start_of_frame() {
    if (!input_wait_until(HIGH)) return EOS;
    if (!input_wait_until(LOW)) return EOS;
    if (!input_skip(BT / 2)) return EOS;
    // now we are exactly at the start of the first bit of the start delimiter
    return NULL;
}

// 3.3.1.2 Bit encoding
error_t read_symbol(symbol_t *r) {
    if (!input_skip(BT / 4)) return EOS;
    bool v1 = input_get();
    if (!input_skip(BT / 2)) return EOS;
    bool v2 = input_get();
    if (!input_skip(BT / 4)) return EOS;

    if (!v1 && v2) *r = BIT_0;
    if (v1 && !v2) *r = BIT_1;
    if (v1 && v2) *r = NH;
    if (!v1 && !v2) *r = NL;

    return NULL;
}

// 3.3.1.2 Bit encoding
error_t read_bit(bool *r) {
    if (!input_skip(BT / 4)) return EOS;
    bool v = input_get();
    if (!input_wait_until(!v)) return EOS;
    if (!input_skip(BT / 2)) return EOS;

    *r = v;

    return NULL;
}

const error_t ERR_MASTER_START_DELIMITER = "failed reading master start delimiter";
const error_t ERR_SLAVE_START_DELIMITER = "failed reading slave start delimiter";

#define read_symbol_expect(e) do {\
        err = read_symbol(&s);\
        if (err) return err;\
        if (s != e) return ERR_MASTER_START_DELIMITER;\
    } while(0)

#define read_bit_expect(e) do {\
        err = read_bit(&bit);\
        if (err) return err;\
        if (bit != e) return ERR_MASTER_START_DELIMITER;\
    } while(0)

// 3.3.1.5 Start delimiter
error_t read_start_delimiter(bool *is_master) {
    symbol_t s;
    bool bit;
    error_t err = read_symbol(&s);
    if (err) return err;
    if (s == NH) {
        // master: "NH", "NL", "0", "NH, "NL", "0", "0", "0"
        *is_master = true;
        read_symbol_expect(NL);
        read_bit_expect(false);
        read_symbol_expect(NH);
        read_symbol_expect(NL);
        read_bit_expect(false);
        read_bit_expect(false);
        read_bit_expect(false);
        return NULL;
    }
    if (s == BIT_1) {
        // slave: "1", "1", "1", "NL, "NH", "1", "NL", "NH"
        *is_master = false;
        read_bit_expect(true);
        read_bit_expect(true);
        read_symbol_expect(NL);
        read_symbol_expect(NH);
        read_bit_expect(true);
        read_symbol_expect(NL);
        read_symbol_expect(NH);
        return NULL;
    }
    return "failed reading start delimiter";
}

// 3.3.1.6 End Delimiter
error_t read_end_delimiter() {
    symbol_t s;
    error_t err = read_symbol(&s);
    if (err) return err;
    if (s != NL) return "failed reading end delimiter";
    return NULL;
}

error_t read_byte(uint8_t *r) {
    *r = 0;
    for (int i = 0; i < 8; i++) {
        bool bit;
        error_t err = read_bit(&bit);
        if (err) return err;
        if (bit) *r |= 1;
        *r <<= 1;
    }
    return NULL;
}

error_t read_word(uint16_t *r) {
    *r = 0;
    for (int i = 0; i < 16; i++) {
        bool bit;
        error_t err = read_bit(&bit);
        if (err) return err;
        *r <<= 1;
        if (bit) *r |= 1;
    }
    return NULL;
}

error_t read_bytes(uint8_t r[], int n) {
    for (int i = 0; i < n; i++) {
        error_t err = read_byte(&r[i]);
        if (err) return err;
    }
    return NULL;
}

error_t read_words(uint16_t r[], int n) {
    for (int i = 0; i < n; i++) {
        error_t err = read_word(&r[i]);
        if (err) return err;
    }
    return NULL;
}

// 3.4.1.3 Check Sequence
error_t check_crc(uint16_t data[], int n, uint8_t cs) {
    // TODO
    return NULL;
}

struct {
    uint8_t fcode;
    uint16_t address;
} master_frame;

// 3.4.1.1 Master Frame format
// 3.5.2.1 Master Frame format
error_t read_master() {
    wait_until_start_of_frame();

    bool is_master;
    error_t err = read_start_delimiter(&is_master);
    if (err) return err;
    if (!is_master) {
        return "expected master frame, got slave\n";
    }

    uint16_t data;
    err = read_word(&data);

    uint8_t cs;
    err = read_byte(&cs);
    if (err) return err;

    err = check_crc(&data, 1, cs);
    if (err) return err;

    master_frame.fcode = data >> 12;
    master_frame.address = data & 0x0fff;
    return read_end_delimiter();
}

struct {
    uint16_t data[16];
    int size;
} slave_frame;

// 3.4.1.2 Slave Frame format
// 3.5.3.1 Slave Frame format
error_t read_slave(const fcode_t *fcode) {
    wait_until_start_of_frame();

    bool is_master;
    error_t err = read_start_delimiter(&is_master);
    if (err) return err;
    if (is_master) {
        return "expected slave frame, got master\n";
    }

    int remaining = fcode->slave_frame_size / 16;
    slave_frame.size = remaining;
    uint16_t *data = slave_frame.data;
    while (remaining > 0) {
        // one check sequence every 4 words
        int n = remaining <= 4 ? remaining : 4;
        err = read_words(data, n);
        if (err) return err;

        uint8_t cs;
        err = read_byte(&cs);
        if (err) return err;

        err = check_crc(data, n, cs);
        if (err) return err;

        data += n;
        remaining -= n;
    }
    return read_end_delimiter();
}

error_t read_master_slave() {
    error_t err = read_master();
    if (err) return err;
    return read_slave(&fcodes[master_frame.fcode]);
}

void print_master() {
    printf("MASTER [ %d ] -> { 0x%03x } ", master_frame.fcode, master_frame.address);
}

void print_slave() {
    printf("  SLAVE ");
    for (int i = 0; i < slave_frame.size; i++) {
        printf("%04x", slave_frame.data[i]);
    }
    printf("\n");
}

int main(int argc, char *argv[]) {
    FILE *f = fopen(argv[1], "rb");
    input_init(f, argv[2][0] == '1');

    while (1) {
        error_t err = read_master_slave();
        if (err == EOS) {
            break;
        }
        if (err != NULL) {
            //printf("%zd %f %s\n", input_n(), input_t(), err);
            //return 1;
        } else {
            print_master();
            print_slave();
        }
    }

    fclose(f);
    return 0;
}
