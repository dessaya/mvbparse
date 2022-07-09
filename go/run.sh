#!/usr/bin/bash

vars=(
    002:0:6 "fecha y hora"
    003:8:9 "tension de red" # (0-FFH corresponde 0-1500V)

    # Valor de aire de temperatura del sensor de retorno (la temperatura real de la habitación, 1 = 0,1 ℃, valor negativo se muestra con código complemento )
    014:0:2 "temp retorno TC1"
    024:0:2 "temp retorno M1"
    034:0:2 "temp retorno M2"
    044:0:2 "temp retorno T3"
    054:0:2 "temp retorno M1"
    064:0:2 "temp retorno M2"
    074:0:2 "temp retorno TC2"
    084:0:2 "temp retorno M4"
    094:0:2 "temp retorno M3"

    # Valor de aire de temperatura del sensor del viento -byte en valor bajo (la temperatura real de la habitación, 1 = 0,1 ℃, valor negativo se muestra con código complemento)
    014:2:4 "temp viento TC1"
    024:2:4 "temp viento M1"
    034:2:4 "temp viento M2"
    044:2:4 "temp viento T3"
    054:2:4 "temp viento M1"
    064:2:4 "temp viento M2"
    074:2:4 "temp viento TC2"
    084:2:4 "temp viento M4"
    094:2:4 "temp viento M3"

    # Posición de flaps de ventilación
    014:6:7 "flaps ventilacion TC1"
    024:6:7 "flaps ventilacion M1"
    034:6:7 "flaps ventilacion M2"
    044:6:7 "flaps ventilacion T3"
    054:6:7 "flaps ventilacion M1"
    064:6:7 "flaps ventilacion M2"
    074:6:7 "flaps ventilacion TC2"
    084:6:7 "flaps ventilacion M4"
    094:6:7 "flaps ventilacion M3"

    # tension del bus 380v
    015:9:10 "tension bus 380v TC1"
    075:9:10 "tension bus 380v TC2"

    # señales de alarma mitshubishi
    025:0:6 "alarma mitsubishi M1"
    035:0:6 "alarma mitsubishi M2"
    055:0:6 "alarma mitsubishi M1"
    065:0:6 "alarma mitsubishi M2"
    085:0:6 "alarma mitsubishi M4"
    095:0:6 "alarma mitsubishi M3"

    # corriente del motor (1000A / 256)
    015:27:28 "corriente motor TC1"
    025:27:28 "corriente motor M1"
    035:27:28 "corriente motor M2"
    045:27:28 "corriente motor T3"
    055:27:28 "corriente motor M1"
    065:27:28 "corriente motor M2"
    075:27:28 "corriente motor TC2"
    085:27:28 "corriente motor M4"
    095:27:28 "corriente motor M3"

    # estimacion cantidad de pasajeros
    006:10:12 "carga TC1"
    006:12:14 "carga M1"
    006:14:16 "carga M2"
    006:16:18 "carga T3"
    006:18:20 "carga M1"
    006:20:22 "carga M2"
    006:22:24 "carga M3"
    006:24:26 "carga M4"
    006:26:28 "carga TC2"
)

exec go run record/main.go -v -high=02 -low=00 "${vars[@]}" </tmp/fifo
