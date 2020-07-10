from __future__ import annotations
from dataclasses import dataclass
from enum import Enum, unique
import sys
import struct

BT = 666.7e-9
SAMPLE_RATE = 12000000

def to_int(bits):
    return eval('0b' + bits)

# Table 53
@unique
class AddressType(Enum):
    NONE = 0
    LOGICAL = 1
    DEVICE = 2
    ALL_DEVICES = 3
    DEVICE_GROUP = 4

@unique
class MasterRequest(Enum):
    PROCESS_DATA = 0
    RESERVED = 1
    MASTERSHIP_TRANSFER = 2
    GENERAL_EVENT = 3 # parameters
    MESSAGE_DATA = 4
    GROUP_EVENT = 5
    SINGLE_EVENT = 6
    DEVICE_STATUS = 7

@unique
class SlaveFrameSource(Enum):
    NONE = 0
    SINGLE = 1
    PROPOSED_MASTER = 2
    DEVICE_GROUP = 3
    SUBSCRIBED_SOURCE = 4

@unique
class SlaveResponse(Enum):
    NONE = 0
    PROCESS_DATA = 1
    MASTERSHIP_TRANSFER = 2
    EVENT_IDENTIFIER = 3
    MESSAGE_DATA = 4
    DEVICE_STATUS = 5

@unique
class SlaveFrameDestination(Enum):
    NONE = 0
    SUBSCRIBED_SINKS = 1
    MASTER = 2
    SELECTED_DEVICES = 3
    MASTER_OR_MONITOR = 4

@dataclass
class FCode:
    n: int
    address_type: AddressType
    master_request: MasterRequest
    slave_frame_source: SlaveFrameSource
    slave_frame_size: int
    slave_response: SlaveResponse
    slave_frame_destination: SlaveFrameDestination

fcodes = {
    0: FCode(0, AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 16, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    1: FCode(1, AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 32, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    2: FCode(2, AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 64, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    3: FCode(3, AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 128, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    4: FCode(4, AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 256, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    5: FCode(5, AddressType.NONE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    6: FCode(6, AddressType.NONE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    7: FCode(7, AddressType.NONE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    8: FCode(8, AddressType.DEVICE, MasterRequest.MASTERSHIP_TRANSFER, SlaveFrameSource.PROPOSED_MASTER, 16, SlaveResponse.MASTERSHIP_TRANSFER, SlaveFrameDestination.MASTER),
    9: FCode(9, AddressType.ALL_DEVICES, MasterRequest.GENERAL_EVENT, SlaveFrameSource.DEVICE_GROUP, 16, SlaveResponse.EVENT_IDENTIFIER, SlaveFrameDestination.MASTER),
    10: FCode(10, AddressType.DEVICE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    11: FCode(11, AddressType.DEVICE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    12: FCode(12, AddressType.DEVICE, MasterRequest.MESSAGE_DATA, SlaveFrameSource.SINGLE, 256, SlaveResponse.MESSAGE_DATA, SlaveFrameDestination.SELECTED_DEVICES),
    13: FCode(13, AddressType.DEVICE_GROUP, MasterRequest.GROUP_EVENT, SlaveFrameSource.DEVICE_GROUP, 16, SlaveResponse.EVENT_IDENTIFIER, SlaveFrameDestination.MASTER),
    14: FCode(14, AddressType.DEVICE, MasterRequest.SINGLE_EVENT, SlaveFrameSource.SINGLE, 16, SlaveResponse.EVENT_IDENTIFIER, SlaveFrameDestination.MASTER),
    15: FCode(15, AddressType.DEVICE, MasterRequest.DEVICE_STATUS, SlaveFrameSource.SINGLE, 16, SlaveResponse.DEVICE_STATUS, SlaveFrameDestination.MASTER_OR_MONITOR),
}

@dataclass
class MasterFrame:
    fcode: FCode
    address: int

    def is_master(self): return True

    def __str__(self):
        return f'MASTER [f_code={self.fcode.n} {self.fcode.master_request.name}] -> {self.describe_address()}'

    def describe_address(self):
        if self.fcode.address_type == AddressType.LOGICAL:
            return f'[port 0x{self.address:03x}]'
        return f'[physical 0x{self.address:03x}]'

def parse_master_frame(data, previous_frame):
    # 3.4.1.1 Master Frame Format
    assert len(data) == 3
    check_crc(data[:2], data[2])

    # 3.5.2.1 Master Frame format
    fcode = fcodes[to_int(data[0][:4])]
    address = to_int(data[0][4:] + data[1])
    return MasterFrame(fcode, address)

@dataclass
class SlaveFrame:
    data: list[str]

    def is_master(self): return False

    def __str__(self):
        return f'SLAVE {len(self.data)} bytes'

# 3.4.1.2 Slave Frame Format
slave_formats = {
    # #bytes -> CRC locations
    3: [(0, 2)],
    5: [(0, 4)],
    9: [(0, 8)],
    18: [(0, 8), (9, 17)],
    36: [(0, 8), (9, 17), (18, 26), (27, 35)],
}
def parse_slave_frame(data: list[str], master_frame: MasterFrame):
    data_ = []
    for a, crc in slave_formats[len(data)]:
        check_crc(data[a:crc], data[crc])
        data_.extend(data[a:crc])
    assert len(data_) == master_frame.fcode.slave_frame_size / 8
    parser = slave_responses.get(master_frame.fcode.master_request, None)
    if parser:
        return parser(data_, master_frame)
    return SlaveFrame(data_)

@dataclass
class ProcessDataResponse:
    logical_address: int
    data: list[int]

    def is_master(self): return False

    def __str__(self):
        n = len(self.data)
        return 'SLAVE ProcessDataResponse {} bytes: 0x{:0{w}x}'.format(n, to_int(''.join(self.data)), w=n * 2)


# 3.5.4.1 Process Data telegram
def parse_slave_frame_process_data(data: list[str], master_frame: MasterFrame):
    return ProcessDataResponse(master_frame.address, data)

@dataclass
class MessageDataResponse:
    device_address: int
    data: list[int]

    def is_master(self): return False

    def __str__(self):
        return f'SLAVE MessageDataResponse {len(self.data)} bytes'

# 3.5.4.2 Message Data
def parse_slave_frame_message_data(data: list[str], master_frame: MasterFrame):
    return MessageDataResponse(master_frame.address, data)

@dataclass
class DeviceStatusResponse:
    device_address: int
    SP: str
    BA: str
    GW: str
    MD: str
    class_specific: str
    LAT: str
    RLD: str
    SSD: str
    SDD: str
    ERD: str
    FRC: str
    DNR: str
    SER: str

    def is_master(self): return False

    def __str__(self):
        return f'SLAVE {repr(self)}'

# 3.6.4.1.1 Device_Status
def parse_slave_frame_device_status(data: list[str], master_frame: MasterFrame):
    assert len(data) == 2
    SP, BA, GW, MD = data[0][:4]
    class_specific = data[0][4:]
    LAT, RLD, SSD, SDD, ERD, FRC, DNR, SER = data[1]
    return DeviceStatusResponse(master_frame.address, SP, BA, GW, MD, class_specific, LAT, RLD, SSD, SDD, ERD, FRC, DNR, SER)

# 3.5.4 Telegram types
slave_responses = {
    MasterRequest.PROCESS_DATA: parse_slave_frame_process_data,
    MasterRequest.MESSAGE_DATA: parse_slave_frame_message_data,
    MasterRequest.DEVICE_STATUS: parse_slave_frame_device_status,
}


# 3.4.1.3 Check Sequence
def check_crc(data, crc):
    assert len(data) in (2, 4, 8)
    # TODO not working
    return
    print(f'2 {data=}')
    data = divide_mod(data, '11100101')
    print(f'5 {data=}')
    ones = len([b for b in data if b == '1'])
    print(f'6 {data=}')
    data = data + '0' if ones % 2 == 0 else '1'
    print(f'7 {data=}')
    data = data.rjust(8, '0')
    print(f'8 {data=}')
    data = ''.join('1' if b == '0' else '0' for b in data)
    print(f'9 {data=} {crc=}')
    assert data == crc

def divide_mod(data, div):
    data = data + '0' * 7
    res = list(data)
    for i in range(len(data) - 7):
        if res[i] == '0':
            continue
        if all(b == '0' for b in res[:-7]):
            break
        for j in range(len(div)):
            res[i + j] = xor(res[i], div[j])
    return ''.join(res[-7:])

def xor(a, b):
    x = a == '1' and b == '0' or a == '0' and b == '1'
    return '1' if x else '0'

assert check_crc(['01111110', '11000011'], '11011101') == None

frame_types = {
    ('NH', 'NL', '0', 'NH', 'NL', '0', '0', '0'): parse_master_frame,
    ('1', '1', '1', 'NL', 'NH', '1', 'NL', 'NH'): parse_slave_frame,
}

# 3.3.1.7 Valid frame
def read_frame(stream, previous_frame):
    t, v = stream.find(1)

    # 3.3.1.4 Start Bit
    start = t
    start_bit, t, v = read_bit(stream, start, t, v)
    assert(start_bit == '1')
    i = 0
    data = []
    while True:
        isStartDelimiter = i == 0
        byte, t, v = read_byte(stream, start + BT + i * 8 * BT, t, v, isStartDelimiter)
        if not byte:
            break
        if isStartDelimiter:
            # 3.3.1.5 Start Delimiter
            start_delimiter = tuple(byte)
            assert start_delimiter in frame_types
            frame_parser = frame_types[start_delimiter]
        else:
            data.append(''.join(byte))
        i += 1
    frame = frame_parser(data, previous_frame)
    return start, frame

def read_byte(stream, start, t, v, isStartDelimiter):
    bits = []
    for i in range(8):
        bit, t, v = read_bit(stream, start + i * BT, t, v)
        if not isStartDelimiter and bit != '1' and bit != '0':
            # 3.3.1.6 End Delimiter
            assert i == 0
            assert bit == 'NL'
            bit, t, v = read_bit(stream, start + (i + 1) * BT, t, v)
            assert(bit == 'NH')
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
        assert i >= self.sample_i
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
        t = self.sample_i / SAMPLE_RATE

        if self.sample_i % SAMPLE_RATE == 0:
            sys.stderr.write(f'{t=}\n')
        self.sample_i += 1

        return t, v

def main():
    n = int(sys.argv[2])

    stream = Stream()
    t, v = stream.next()
    assert v == 0
    previous_frame = None
    while True:
        try:
            t, frame = read_frame(stream, previous_frame)
            if previous_frame and previous_frame.is_master():
                if frame.is_master():
                    print(f'{t=:.6f} :: {str(previous_frame)} :: (no slave frame)')
                else:
                    print(f'{t=:.6f} :: {str(previous_frame)} :: {str(frame)}')
            previous_frame = frame
            n -= 1
            if n == 0:
                break
        except AssertionError as e:
            sys.stderr.write(f"AssertionError around {t=}: {str(e)}\n")
            previous_frame = None
try:
    main()
except StopIteration:
    pass
