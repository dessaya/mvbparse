#include "sapi.h"

#define PININT_INDEX 2
#define INPUT_PORT 2
#define INPUT_PIN 0

#define READ() Chip_GPIO_ReadPortBit(LPC_GPIO_PORT, INPUT_PORT, INPUT_PIN)

// BT / 4 = 166 ns at 204 MHz
#define BT4_CYCLES 34
// BT / 2
#define BT2_CYCLES (BT4_CYCLES * 2)
// BT * 3/4
#define BT34_CYCLES (3 * BT4_CYCLES)
// BT
#define BT_CYCLES (BT2_CYCLES * 2)

// assume line is idle when no edge is detected within 2 * BT
#define IDLE_CYCLES (BT_CYCLES * 2)

// 3.3.1.2 Bit encoding
typedef enum {
    BIT_0, // LOW  HIGH
    BIT_1, // HIGH LOW
    NH,    // HIGH HIGH
    NL,    // LOW  LOW
} symbol_t;

typedef struct {
    symbol_t syms[1024];
    int size;
    bool ready;
    bool error;
} rxBuf_t;

#define RXBUFS_SIZE 10

// circular buffer
rxBuf_t rxBufs[RXBUFS_SIZE] = {0};

static inline bool waitUntilElapsedOrEdge(int cycles, bool v1) {
    uint32_t start = cyclesCounterRead();
    uint32_t end = start + cycles;
    do {
        bool v2 = READ();
        if (v2 != v1) {
            // edge detected before wait time elapsed
            return v2;
        }
    } while (cyclesCounterRead() < end);
    // edge not detected
    return v1;
}

static inline bool waitUntilHigh() {
    for (int i = 0; i < IDLE_CYCLES; i++) {
        if (!READ()) return true;
    }
    return false;
}

static inline bool readSymbol(symbol_t *s, bool v1) {
    // now we are at BT / 4; wait until BT * 3 / 4
    bool v2 = waitUntilElapsedOrEdge(BT2_CYCLES, v1);
    if (v2 != v1) {
        // edge detected; we should be at BT / 2; wait for BT * 3/4
        bool v3 = waitUntilElapsedOrEdge(BT34_CYCLES, v2);
        *s = v2 ? BIT_0 : BIT_1;
        return v3;
    }
    // edge not detected; we should be at BT * 3 / 4; wait for BT / 2
    bool v3 = waitUntilElapsedOrEdge(BT2_CYCLES, v2);
    *s = v2 ? NH : NL;
    return v3;
}

void GPIO2_IRQHandler(void) {
    static int rxBufIdx = 0;

    rxBuf_t *rxBuf = &rxBufs[rxBufIdx];
    if (rxBuf->ready) {
        printf("rx buffer is full\r\n");
        return;
    }
    rxBuf->size = 0;
    rxBuf->error = false;

    // 3.3.1.5 Start Delimiter
    // wait until the start of the first symbol of the start delimiter
    if (!waitUntilHigh()) {
        rxBuf->error = true;
        goto end;
    }

    // readSymbol() expects to start from BT / 4
    bool v = waitUntilElapsedOrEdge(BT4_CYCLES, true);
    if (!v) {
        // edge detected -- should not happen
        rxBuf->error = true;
        goto end;
    }

    while (1) {
        symbol_t *s = &rxBuf->syms[rxBuf->size++];
        v = readSymbol(s, v);
        // 3.3.1.6 End Delimiter
        if (rxBuf->size > 8 && (*s == NH || *s == NL)) {
            goto end;
        }
    }

end:
    rxBuf->ready = true;
    rxBufIdx = (rxBufIdx + 1) % RXBUFS_SIZE;
	Chip_PININT_ClearIntStatus(LPC_GPIO_PIN_INT, PININTCH(PININT_INDEX));
}

rxBuf_t *receiveFrame() {
    static int rxBufIdx = 0;
    rxBuf_t * rxBuf = &rxBufs[rxBufIdx];
    while (!rxBuf->ready)
        ;
    rxBufIdx = (rxBufIdx + 1) % RXBUFS_SIZE;
    return rxBuf;
}

void printFrame(rxBuf_t *rxBuf) {
    if (rxBuf->error) {
        printf("[error] ");
    } else {
        printf("[OK] ");
    }
    for (int i = 0; i < rxBuf->size; i++) {
        switch (rxBuf->syms[i]) {
            case NH: printf("NH "); break;
            case NL: printf("NL "); break;
            case BIT_0: printf("0 "); break;
            case BIT_1: printf("1 "); break;
        }
    }
    printf("\r\n");
}

int main(void) {
    boardInit();
    uartConfig(UART_USB, 115200);

	// inicialización interrupción TFIL0 = GPIO2[0]
	gpioInit(T_FIL0, GPIO_INPUT_PULLDOWN);

	/* Configure interrupt channel for the GPIO pin in SysCon block */
	Chip_SCU_GPIOIntPinSel(PININT_INDEX, INPUT_PORT, INPUT_PIN);

	/* Configure channel interrupt as edge sensitive and falling edge interrupt */
	Chip_PININT_ClearIntStatus(LPC_GPIO_PIN_INT, PININTCH(PININT_INDEX));
	Chip_PININT_SetPinModeEdge(LPC_GPIO_PIN_INT, PININTCH(PININT_INDEX));
	Chip_PININT_EnableIntLow(LPC_GPIO_PIN_INT, PININTCH(PININT_INDEX));

	/* Enable interrupt in the NVIC */
	NVIC_ClearPendingIRQ(PIN_INT2_IRQn);
	NVIC_EnableIRQ(PIN_INT2_IRQn);

	printf("Init OK\r\n");

	while (true) {
		rxBuf_t *rxBuf = receiveFrame();
        printFrame(rxBuf);
        rxBuf->ready = false;
	}
}
