from __future__ import annotations
from dataclasses import dataclass
from enum import Enum, unique
import sys
import struct

BT = 666.7e-9
SAMPLE_RATE = 12000000

def to_int(bits):
    return eval('0b' + bits)

frame_types = {
    ('NH', 'NL', '0', 'NH', 'NL', '0', '0', '0'): True,  # master
    ('1', '1', '1', 'NL', 'NH', '1', 'NL', 'NH'): False, # slave
}

@dataclass
class Frame:
    t: float
    is_master: bool
    data: list[int]

    def __str__(self):
        return ''.join(f'{x:02x}' for x in self.data)

# 3.3.1.7 Valid frame
def read_frame(stream):
    t, v = stream.find(1)

    # 3.3.1.4 Start Bit
    start = t
    start_bit, t, v = read_bit(stream, start, t, v)
    assert(start_bit == '1'), "start bit should be 1"
    i = 0
    data = []
    is_master = None
    while True:
        isStartDelimiter = i == 0
        byte, t, v = read_byte(stream, start + BT + i * 8 * BT, t, v, isStartDelimiter)
        if not byte:
            # end delimiter
            break
        if isStartDelimiter:
            # 3.3.1.5 Start Delimiter
            start_delimiter = tuple(byte)
            assert start_delimiter in frame_types, "start_delimiter"
            is_master = frame_types[start_delimiter]
        else:
            data.append(to_int(''.join(byte)))
        i += 1
    assert is_master is not None, "no start delimiter found"
    assert len(data) > 0, "no data"
    return Frame(start, is_master, data)

def read_byte(stream, start, t, v, isStartDelimiter):
    bits = []
    for i in range(8):
        bit, t, v = read_bit(stream, start + i * BT, t, v)
        if not isStartDelimiter and bit != '1' and bit != '0':
            # 3.3.1.6 End Delimiter
            assert i == 0, "unexpected end delimiter"
            assert bit == 'NL', "end delimiter: expected NL"
            bit, t, v = read_bit(stream, start + (i + 1) * BT, t, v)
            assert bit == 'NH', "end delimiter: expected NH"
            _, t, v = read_bit(stream, start + (i + 2) * BT, t, v)
            return None, t, v
        bits.append(bit)
    return bits, t, v

bit_names = {
    # 3.3.1.2 Bit encoding
    (1, 0): '1',
    (0, 1): '0',
    # 3.3.1.3 Non-data symbols
    (1, 1): 'NH',
    (0, 0): 'NL',
}

def read_bit(stream, start, t, v):
    t, v1 = stream.skip_until(start + BT / 4)
    t, v2 = stream.skip_until(start + 3 * BT / 4)
    t, v = stream.skip_until(start + BT)
    return bit_names[(v1, v2)], t, v

class Stream:
    def __init__(self):
        self.f = open(sys.argv[1], 'rb')
        self.sample_i = 0
        self.block = None
        self.block_len = 0
        self.block_i = 0

    def next_block(self):
        self.block = self.f.read(4096)
        self.block_i += self.block_len
        self.block_len = len(self.block)

    def check_block(self):
        while self.block == None or self.sample_i >= self.block_i + self.block_len:
            self.next_block()

    def next_sample(self):
        self.check_block()
        b = self.block[self.sample_i - self.block_i]
        return b

    def skip_until(self, until):
        i = int(until * SAMPLE_RATE)
        assert i >= self.sample_i, "skip to the past?"
        self.sample_i = i
        return self.next()

    def find(self, v):
        self.check_block()
        while True:
            i = self.block.find(b'\0' if v else b'\2', max(0, self.sample_i - self.block_i))
            if i > 0:
                self.sample_i = self.block_i + i
                return self.next()
            self.next_block()

    def next(self, until=None):
        sample = self.next_sample()
        v = 0 if sample == 0x02 else 1 # señal invertida
        t = self.time()
        self.sample_i += 1

        return t, v

    def time(self):
        return self.sample_i / SAMPLE_RATE


def main():
    stream = Stream()
    t, v = stream.next()
    assert v == 0

    previous_frame = None
    while True:
        try:
            frame1 = previous_frame if previous_frame else read_frame(stream)
            previous_frame = None
            assert frame1.is_master, "expected master frame"

            frame2 = read_frame(stream)
            if frame2.is_master:
                previous_frame = frame2
                frame2 = None

            print(f'{frame1.t},{str(frame1)},{"" if not frame2 else str(frame2)}')
        except AssertionError as e:
            sys.stderr.write(f"t={stream.time():.6f}s :: AssertionError: {str(e)}\n")
            previous_frame = None
try:
    main()
except StopIteration:
    pass
