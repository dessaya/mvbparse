# gendata.py: lee de un CSV creado a partir de mvb_signal.py, y escupe los
# datos en el formato necesario para gen.c

import sys

def b(a):
    return ', '.join(f"0x{a[i:i+2]}" for i in range(0, len(a), 2))

for line in sys.stdin:
    t, m, s = line.strip().split(',')
    print(f"{{ {{ {b(m)} }}, (uint8_t[]){{ {b(s)} }}, {len(s) // 2} }},")
