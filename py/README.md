# Software para decodificación de tramas MVB

`mvb_signal.py`: Lee de stdin un flujo de bytes donde cada byte representa una
muestra de la señal de entrada. Produce como salida un CSV con los bytes
decodificados de cada telegrama con formato `<time>,<master>,<slave>`. Ejemplo:

```
$ python3 mvb_signal.py < captura.bin
0.00017633333333333333,4390d6,971e000000821406df1e0b310f0017058cf8000000000000034dc9119411a811a8040588
0.0011865833333333333,431bf7,30000f0c011000000f00000000000011a8100000000000000000ff0000000000000000ff
0.0021969166666666665,000134,971e07
0.002248,4010c5,04004830580048808f3bf000001bf91bf9452b00000000000000690000000000000000ff
```

`mvbparse.py`: Lee de stdin la salida de `mvb_signal.py` y produce como salida
la evolución de las variables MVB.
