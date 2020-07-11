from __future__ import annotations
from dataclasses import dataclass
from enum import Enum, unique
import sys
import struct

def to_hex(data: list[int]) -> str:
    return '0x' + ''.join(f'{x:02x}' for x in data)

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
    t: float
    fcode: FCode
    address: int

    def __str__(self):
        return f'MASTER [{self.fcode.master_request.name}] -> {self.describe_address()}'

    def describe_address(self):
        if self.fcode.address_type == AddressType.LOGICAL:
            return f'[port 0x{self.address:03x}]'
        return f'[physical 0x{self.address:03x}]'

def parse_master_frame(t, data):
    # 3.4.1.1 Master Frame Format
    assert len(data) == 3, "master frame: len(data) == 3"
    check_crc(data[:2], data[2])

    # 3.5.2.1 Master Frame format
    fcode = fcodes[(data[0] & 0xf0) >> 4]
    address = (data[0] & 0x0f) << 8 | data[1]
    return MasterFrame(t, fcode, address)

@dataclass
class SlaveFrame:
    data: list[int]

    def __str__(self):
        return f'SLAVE ?? ({len(self.data):2}b)'

# 3.4.1.2 Slave Frame Format
slave_formats = {
    # #bytes -> CRC locations
    3: [(0, 2)],
    5: [(0, 4)],
    9: [(0, 8)],
    18: [(0, 8), (9, 17)],
    36: [(0, 8), (9, 17), (18, 26), (27, 35)],
}
def parse_slave_frame(data: list[int], master_frame: MasterFrame):
    data_ = []
    for a, crc in slave_formats[len(data)]:
        check_crc(data[a:crc], data[crc])
        data_.extend(data[a:crc])
    assert len(data_) == master_frame.fcode.slave_frame_size / 8, "slave frame: invalid data length"
    parser = slave_responses.get(master_frame.fcode.master_request, None)
    if parser:
        return parser(data_, master_frame)
    return SlaveFrame(data_)

class ProcessVariable:
    def __init__(self, port):
        self.port = port
        self.data = []
        self.n = 0

    def update(self, t, data):
        if not self.data or self.data[-1][1] != data:
            self.data.append((t, data))
        self.n += 1

    def __repr__(self):
        port, n, data = self.port, self.n, self.data
        return f'[[{port=}] [{n=:>5}] [{self.format_changes()}]]'

    def format_changes(self):
        if len(self.data) > 18:
            return '... ' + str(len(self.data)) + ' ...'
        return ' '.join(f'\n                  t={t:7.3f}s  {to_hex(data)}' for t, data in self.data)

    def __lt__(self, v):
        return len(self.data) < len(v.data)

variables = {
}

@dataclass
class ProcessDataResponse:
    t: float
    port: int
    data: list[int]

    def __post_init__(self):
        port = f'0x{self.port:03x}'
        if port not in variables:
            variables[port] = ProcessVariable(port)
        variables[port].update(self.t, self.data)

    def __str__(self):
        n = len(self.data)
        return f'SLAVE ({n:2}b): {to_hex(self.data)}'


# 3.5.4.1 Process Data telegram
def parse_slave_frame_process_data(data: list[int], master_frame: MasterFrame):
    return ProcessDataResponse(master_frame.t, master_frame.address, data)

@dataclass
class MessageDataResponse:
    device_address: int
    data: list[int]

    def __str__(self):
        return f'SLAVE MessageDataResponse {len(self.data)} bytes'

# 3.5.4.2 Message Data
def parse_slave_frame_message_data(data: list[int], master_frame: MasterFrame):
    return MessageDataResponse(master_frame.address, data)

@dataclass
class DeviceStatusResponse:
    device_address: int
    SP: int
    BA: int
    GW: int
    MD: int
    class_specific: list(int)
    LAT: int
    RLD: int
    SSD: int
    SDD: int
    ERD: int
    FRC: int
    DNR: int
    SER: int

    def __str__(self):
        return f'SLAVE {repr(self)}'

def to_bits(n: int):
    assert n < 256, "??"
    return (
        1 if n & 0b10000000 else 0,
        1 if n & 0b01000000 else 0,
        1 if n & 0b00100000 else 0,
        1 if n & 0b00010000 else 0,
        1 if n & 0b00001000 else 0,
        1 if n & 0b00000100 else 0,
        1 if n & 0b00000010 else 0,
        1 if n & 0b00000001 else 0,
    )

# 3.6.4.1.1 Device_Status
def parse_slave_frame_device_status(data: list[int], master_frame: MasterFrame):
    assert len(data) == 2, "parse_slave_frame_device_status len data"
    SP, BA, GW, MD, *class_specific = to_bits(data[0])
    LAT, RLD, SSD, SDD, ERD, FRC, DNR, SER = to_bits(data[1])
    return DeviceStatusResponse(master_frame.address, SP, BA, GW, MD, class_specific, LAT, RLD, SSD, SDD, ERD, FRC, DNR, SER)

# 3.5.4 Telegram types
slave_responses = {
    MasterRequest.PROCESS_DATA: parse_slave_frame_process_data,
    MasterRequest.MESSAGE_DATA: parse_slave_frame_message_data,
    MasterRequest.DEVICE_STATUS: parse_slave_frame_device_status,
}


# 3.4.1.3 Check Sequence
def check_crc(data, crc):
    assert len(data) in (2, 4, 8), "data length"
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
    assert data == crc, "CRC"

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

def from_hex(s):
    assert len(s) % 2 == 0, "odd hex length"
    return [int(s[i * 2: i * 2 + 2], 16) for i in range(len(s) // 2)]

def main():
    n = int(sys.argv[2])

    with open(sys.argv[1]) as f:
        while True:
            try:
                line = next(f)
                t, master, slave = line.strip().split(',')

                t = float(t)
                master = parse_master_frame(t, from_hex(master))
                slave = parse_slave_frame(from_hex(slave), master) if slave else 'no slave frame'
                print(f'{t=:.6f} :: {str(master)} :: {str(slave)}')

                n -= 1
                if n == 0:
                    break
            except AssertionError as e:
                sys.stderr.write(f"{t=:.6f}s :: AssertionError: {str(e)}\n")

try:
    main()
except StopIteration:
    pass

import pprint
pprint.pprint(sorted(variables.values()))
