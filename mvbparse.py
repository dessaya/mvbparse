from __future__ import annotations
from dataclasses import dataclass
from enum import Enum
import sys

ns = 1e-9
BT = 666.7 * ns

def to_int(bits):
    return eval('0b' + bits)

# Table 53
class AddressType(Enum):
    NONE = 0
    LOGICAL = 1
    DEVICE = 2
    ALL_DEVICES = 3
    DEVICE_GROUP = 4

class MasterRequest(Enum):
    PROCESS_DATA = 0
    RESERVED = 1
    MASTERSHIP_TRANSFER = 2
    GENERAL_EVENT = 3 # parameters
    MESSAGE_DATA = 4
    GROUP_EVENT = 5
    SINGLE_EVENT = 6
    DEVICE_STATUS = 7

class SlaveFrameSource(Enum):
    NONE = 0
    SINGLE = 1
    PROPOSED_MASTER = 2
    DEVICE_GROUP = 3
    SUBSCRIBED_SOURCE = 4

class SlaveResponse(Enum):
    NONE = 0
    PROCESS_DATA = 1
    MASTERSHIP_TRANSFER = 2
    EVENT_IDENTIFIER = 3
    MESSAGE_DATA = 4
    DEVICE_STATUS = 5

class SlaveFrameDestination(Enum):
    NONE = 0
    SUBSCRIBED_SINKS = 1
    MASTER = 2
    SELECTED_DEVICES = 3
    MASTER_OR_MONITOR = 4

@dataclass
class FCode:
    address_type: AddressType
    master_request: MasterRequest
    slave_frame_source: SlaveFrameSource
    slave_frame_size: int
    slave_response: SlaveResponse
    slave_frame_destination: SlaveFrameDestination

fcodes = {
    0: FCode(AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 16, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    1: FCode(AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 32, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    2: FCode(AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 64, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    3: FCode(AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 128, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    4: FCode(AddressType.LOGICAL, MasterRequest.PROCESS_DATA, SlaveFrameSource.SUBSCRIBED_SOURCE, 256, SlaveResponse.PROCESS_DATA, SlaveFrameDestination.SUBSCRIBED_SINKS),
    5: FCode(AddressType.NONE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    6: FCode(AddressType.NONE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    7: FCode(AddressType.NONE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    8: FCode(AddressType.DEVICE, MasterRequest.MASTERSHIP_TRANSFER, SlaveFrameSource.PROPOSED_MASTER, 16, SlaveResponse.MASTERSHIP_TRANSFER, SlaveFrameDestination.MASTER),
    9: FCode(AddressType.ALL_DEVICES, MasterRequest.GENERAL_EVENT, SlaveFrameSource.DEVICE_GROUP, 16, SlaveResponse.EVENT_IDENTIFIER, SlaveFrameDestination.MASTER),
    10: FCode(AddressType.DEVICE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    11: FCode(AddressType.DEVICE, MasterRequest.RESERVED, SlaveFrameSource.NONE, 0, SlaveResponse.NONE, SlaveFrameDestination.NONE),
    12: FCode(AddressType.DEVICE, MasterRequest.MESSAGE_DATA, SlaveFrameSource.SINGLE, 256, SlaveResponse.MESSAGE_DATA, SlaveFrameDestination.SELECTED_DEVICES),
    13: FCode(AddressType.DEVICE_GROUP, MasterRequest.GROUP_EVENT, SlaveFrameSource.DEVICE_GROUP, 16, SlaveResponse.EVENT_IDENTIFIER, SlaveFrameDestination.MASTER),
    14: FCode(AddressType.DEVICE, MasterRequest.SINGLE_EVENT, SlaveFrameSource.SINGLE, 16, SlaveResponse.EVENT_IDENTIFIER, SlaveFrameDestination.MASTER),
    15: FCode(AddressType.DEVICE, MasterRequest.DEVICE_STATUS, SlaveFrameSource.SINGLE, 16, SlaveResponse.DEVICE_STATUS, SlaveFrameDestination.MASTER_OR_MONITOR),
}

@dataclass
class MasterFrame:
    fcode: FCode
    address: int

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

# 3.5.4.1 Process Data telegram
def parse_slave_frame_process_data(data: list[str], master_frame: MasterFrame):
    return ProcessDataResponse(master_frame.address, data)

@dataclass
class MessageDataResponse:
    device_address: int
    data: list[int]

# 3.5.4.2 Message Data
def parse_slave_frame_message_data(data: list[str], master_frame: MasterFrame):
    return MessageDataResponse(master_frame.address, data)

# 3.5.4 Telegram types
slave_responses = {
    MasterRequest.PROCESS_DATA: parse_slave_frame_process_data,
    MasterRequest.MESSAGE_DATA: parse_slave_frame_message_data,
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
    t, v = next(stream)
    while v == 0:
        t, v = next(stream)

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
    print(f'{start:.6f} {frame=}')
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
    tp, vp = t, v
    while t < start + BT / 4:
        tp, vp, t, v = t, v, *next(stream)
    v1 = vp
    while t < start + 3 * BT / 4:
        tp, vp, t, v = t, v, *next(stream)
    v2 = vp
    while t < start + BT:
        t, v = next(stream)
    return bit_names[(v1, v2)], t, v

def read_events_csv():
    with open(sys.argv[1], 'r') as f:
        next(f)
        while True:
            line = next(f, None)
            if not line:
                break
            t, v = line.strip().split(',')
            t = float(t.strip())
            v = int(v.strip())
            # print(1 - v, t)
            yield t, 1 - v # señal invertida

def read():
    stream = read_events_csv()
    t, v = next(stream)
    assert v == 0
    previous_frame = None
    while True:
        try:
            t, previous_frame = read_frame(stream, previous_frame)
        except AssertionError as e:
            print(f"AssertionError around {t=}: {str(e)}")
            pass

try:
    read()
except StopIteration:
    pass
