#include "sapi.h"
#include "gen.h"

#define TICKRATE_HZ 10

void initSPI() {
	Chip_SCU_PinMuxSet( 0x1, 4, (SCU_MODE_PULLUP | SCU_MODE_FUNC5)); // SSP1_MOSI
	Chip_SCU_PinMuxSet( 0xF, 4, (SCU_MODE_PULLUP | SCU_MODE_FUNC0)); // SSP1_SCK
	Chip_SCU_PinMuxSet( 0x6, 1, (SCU_MODE_PULLUP | SCU_MODE_FUNC0));
	Chip_SSP_Init(LPC_SSP1);
	Chip_SSP_SetFormat(LPC_SSP1, SSP_BITS_8, SSP_FRAMEFORMAT_SPI, SSP_CLOCK_CPHA1_CPOL1);
	Chip_SSP_SetBitRate(LPC_SSP1, 3000000);
	Chip_SSP_Enable(LPC_SSP1);
}

int main(void)
{
    boardInit();
    uartConfig(UART_USB, 115200);
    cyclesCounterInit(EDU_CIAA_NXP_CLOCK_SPEED);

	gpioInit(LED1, GPIO_OUTPUT);
    initSPI();

	printf("Init OK\r\n");

	sendbuf_reset();

	//uint8_t receivedByte;

	while (1) {
		//while (!uartReadByte( UART_USB, &receivedByte ));
        delayInaccurateUs(750);
        sendbuf_t *buf = next_telegram();
        spiWrite(SPI0, buf->data, buf->bytes);
	}
}
