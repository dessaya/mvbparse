package mvb

import (
	"errors"
	"fmt"
	"math/bits"
)

// 3.2.3.1 Signalling speed (bit period in seconds)
const (
	BR = 1_500_000.0
	BT = 1 / BR
)

const (
	BT_SAMPLES   = int(BT * SampleRate)
	BT2_SAMPLES  = int(BT * SampleRate / 2)
	BT4_SAMPLES  = int(BT * SampleRate / 4)
	BT34_SAMPLES = int(3 * BT * SampleRate / 4)
)

const (
	HIGH = true
	LOW  = false
)

type Symbol uint8

// 3.3.1.2 Bit encoding
const (
	BIT_0 = Symbol(iota) // LOW  HIGH
	BIT_1                // HIGH LOW
	NH                   // HIGH HIGH
	NL                   // LOW  LOW
)

func (s Symbol) String() string {
	switch s {
	case BIT_0:
		return "0"
	case BIT_1:
		return "1"
	case NH:
		return "NH"
	case NL:
		return "NL"
	}
	panic("unreachable")
}

type AddressType uint8

const (
	AT_NONE = AddressType(iota)
	AT_LOGICAL
	AT_DEVICE
	AT_ALL_DEVICES
	AT_DEVICE_GROUP
)

type MasterRequest uint8

const (
	MR_PROCESS_DATA = MasterRequest(iota)
	MR_RESERVED
	MR_MASTERSHIP_TRANSFER
	MR_GENERAL_EVENT // parameters
	MR_MESSAGE_DATA
	MR_GROUP_EVENT
	MR_SINGLE_EVENT
	MR_DEVICE_STATUS

	MR_AMOUNT
)

func (m MasterRequest) String() string {
	switch m {
	case MR_PROCESS_DATA:
		return "[00-04] PROCESS_DATA"
	case MR_RESERVED:
		return "[05-07/10-11] RESERVED"
	case MR_MASTERSHIP_TRANSFER:
		return "[08] MASTERSHIP_TRANSFER"
	case MR_GENERAL_EVENT:
		return "[09] GENERAL_EVENT"
	case MR_MESSAGE_DATA:
		return "[12] MESSAGE_DATA"
	case MR_GROUP_EVENT:
		return "[13] GROUP_EVENT"
	case MR_SINGLE_EVENT:
		return "[14] SINGLE_EVENT"
	case MR_DEVICE_STATUS:
		return "[15] DEVICE_STATUS"
	}
	panic("unreachable")
}

type SlaveFrameSource uint8

const (
	SFS_NONE = SlaveFrameSource(iota)
	SFS_SINGLE
	SFS_PROPOSED_MASTER
	SFS_DEVICE_GROUP
	SFS_SUBSCRIBED_SOURCE
)

type SlaveResponse uint8

const (
	SR_NONE = SlaveResponse(iota)
	SR_PROCESS_DATA
	SR_MASTERSHIP_TRANSFER
	SR_EVENT_IDENTIFIER
	SR_MESSAGE_DATA
	SR_DEVICE_STATUS
)

type SlaveFrameDestination uint8

const (
	SFD_NONE = SlaveFrameDestination(iota)
	SFD_SUBSCRIBED_SINKS
	SFD_MASTER
	SFD_SELECTED_DEVICES
	SFD_MASTER_OR_MONITOR
)

type FCode struct {
	N                     uint8
	AddressType           AddressType
	MasterRequest         MasterRequest
	SlaveFrameSource      SlaveFrameSource
	SlaveFrameSize        uint
	SlaveResponse         SlaveResponse
	SlaveFrameDestination SlaveFrameDestination
}

func (f *FCode) String() string {
	return fmt.Sprintf("[fcode %02x]", f.N)
}

type Frame interface {
	IsMaster() bool
}

type MasterFrame struct {
	FCode   uint8
	Address uint16
}

func (m *MasterFrame) IsMaster() bool { return true }

type SlaveFrame struct {
	data []byte
}

func (s *SlaveFrame) IsMaster() bool { return false }

type Telegram struct {
	n      uint64
	Master *MasterFrame
	Slave  *SlaveFrame
}

func (t *Telegram) N() uint64 {
	return t.n
}

func (t *Telegram) IsError() bool {
	return false
}

type Error struct {
	error
	n       uint64
	samples []Sample
}

func (err Error) N() uint64 {
	return err.n
}

func (err Error) IsError() bool {
	return true
}

type Event interface {
	N() uint64
	IsError() bool
}

var fcodes = map[uint8]*FCode{
	0:  {0, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 16, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
	1:  {1, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 32, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
	2:  {2, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 64, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
	3:  {3, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 128, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
	4:  {4, AT_LOGICAL, MR_PROCESS_DATA, SFS_SUBSCRIBED_SOURCE, 256, SR_PROCESS_DATA, SFD_SUBSCRIBED_SINKS},
	5:  {5, AT_NONE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
	6:  {6, AT_NONE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
	7:  {7, AT_NONE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
	8:  {8, AT_DEVICE, MR_MASTERSHIP_TRANSFER, SFS_PROPOSED_MASTER, 16, SR_MASTERSHIP_TRANSFER, SFD_MASTER},
	9:  {9, AT_ALL_DEVICES, MR_GENERAL_EVENT, SFS_DEVICE_GROUP, 16, SR_EVENT_IDENTIFIER, SFD_MASTER},
	10: {10, AT_DEVICE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
	11: {11, AT_DEVICE, MR_RESERVED, SFS_NONE, 0, SR_NONE, SFD_NONE},
	12: {12, AT_DEVICE, MR_MESSAGE_DATA, SFS_SINGLE, 256, SR_MESSAGE_DATA, SFD_SELECTED_DEVICES},
	13: {13, AT_DEVICE_GROUP, MR_GROUP_EVENT, SFS_DEVICE_GROUP, 16, SR_EVENT_IDENTIFIER, SFD_MASTER},
	14: {14, AT_DEVICE, MR_SINGLE_EVENT, SFS_SINGLE, 16, SR_EVENT_IDENTIFIER, SFD_MASTER},
	15: {15, AT_DEVICE, MR_DEVICE_STATUS, SFS_SINGLE, 16, SR_DEVICE_STATUS, SFD_MASTER_OR_MONITOR},
}

// 3.4.1.3 Check Sequence
// https://stackoverflow.com/a/49676373
// TODO use lookup table?
func calcCRC(message []byte) byte {
	poly := byte(0xe5)
	crc := byte(0)
	for i := 0; i < len(message); i++ {
		crc ^= message[i]
		for j := 0; j < 8; j++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ (poly << 1)
			} else {
				crc = crc << 1
			}
		}
	}
	crc &= 0xfe
	crc |= uint8(bits.OnesCount8(crc) % 2)
	return ^crc
}

func checkCRC(data []byte, cs byte) error {
	// 3.4.1.3 Check Sequence
	calculated := calcCRC(data)
	// TODO: parity bit?
	// ok := calculated == cs
	ok := (calculated >> 1) == (cs >> 1)
	if !ok {
		return fmt.Errorf("CRC mismatch: expected %x, got %x", calculated, cs)
	}
	return nil
}

// 3.3.1.5 Start delimiter
var (
	masterStartDelimiter = []Symbol{NH, NL, BIT_0, NH, NL, BIT_0, BIT_0, BIT_0}
	slaveStartDelimiter  = []Symbol{BIT_1, BIT_1, BIT_1, NL, NH, BIT_1, NL, NH}
)

type MVBDecoder struct {
	stream *MVBStream
}

func NewDecoder(stream *MVBStream) *MVBDecoder {
	return &MVBDecoder{
		stream: stream,
	}
}

func (d *MVBDecoder) N() uint64 {
	return d.stream.r.n
}

func (d *MVBDecoder) ReadSymbol() (Symbol, error) {
	v1 := d.stream.V()
	// now we are at BT / 4; wait until BT * 3 / 4
	v2, err := d.stream.WaitUntilElapsedOrEdge(BT2_SAMPLES, v1)
	if v2 != v1 {
		s := BIT_1
		if v2 {
			s = BIT_0
		}
		d.stream.Annotate(s.String())
		// edge detected; we should be at BT / 2; wait for BT * 3/4
		_, err := d.stream.WaitUntilElapsed(BT34_SAMPLES)
		if err != nil {
			return 0, err
		}
		return s, nil
	}
	// edge not detected; we should be at BT * 3 / 4; wait for another BT / 2
	s := NL
	if v2 {
		s = NH
	}
	d.stream.Annotate(s.String())
	_, err = d.stream.WaitUntilElapsed(BT2_SAMPLES)
	if err != nil {
		return 0, err
	}
	return s, nil
}

func (d *MVBDecoder) WaitUntilStartOfFrame() error {
	_, err := d.stream.WaitUntil(HIGH)
	if err != nil {
		return err
	}
	_, err = d.stream.WaitUntil(LOW)
	if err != nil {
		return err
	}
	d.stream.Annotate("S")
	v, err := d.stream.WaitUntilElapsedOrEdge(BT34_SAMPLES, LOW)
	if err != nil {
		return err
	}
	if !v {
		return errors.New("invalid start of frame")
	}
	// read_symbol() expects to start from BT / 4
	_, err = d.stream.WaitUntilElapsed(BT4_SAMPLES)
	return err
}

func (d *MVBDecoder) ReadSymbolExpect(e Symbol) error {
	s, err := d.ReadSymbol()
	if err != nil {
		return err
	}
	if s != e {
		return fmt.Errorf("expected symbol %s, got %s", e, s)
	}
	d.stream.Annotate(s.String())
	return nil
}

func (d *MVBDecoder) ReadBit() (byte, error) {
	s, err := d.ReadSymbol()
	if err != nil {
		return 0, err
	}
	switch s {
	case BIT_0:
		return 0, nil
	case BIT_1:
		return 1, nil
	}
	return 0, fmt.Errorf("expected bit, got %s", s)
}

func (d *MVBDecoder) ReadByte() (byte, error) {
	r := byte(0)
	for i := 0; i < 8; i++ {
		bit, err := d.ReadBit()
		if err != nil {
			return 0, err
		}
		r = (r << 1) | bit
	}
	return r, nil
}

func (d *MVBDecoder) ReadBytes(buf []byte) error {
	for i := 0; i < len(buf); i++ {
		var err error
		buf[i], err = d.ReadByte()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *MVBDecoder) ReadStartDelimiter() (isMaster bool, err error) {
	var s Symbol
	s, err = d.ReadSymbol()
	if err != nil {
		return
	}
	var startDelimiter []Symbol
	switch {
	case s == masterStartDelimiter[0]:
		isMaster = true
		startDelimiter = masterStartDelimiter
	case s == slaveStartDelimiter[0]:
		isMaster = false
		startDelimiter = slaveStartDelimiter
	default:
		return false, fmt.Errorf("invalid start delimiter: %s", s)
	}
	for i := 1; i < len(startDelimiter); i++ {
		err = d.ReadSymbolExpect(startDelimiter[i])
		if err != nil {
			return
		}
	}
	return
}

func (d *MVBDecoder) ReadEndDelimiter() (err error) {
	// 3.3.1.6 End Delimiter
	s, err := d.ReadSymbol()
	if err != nil {
		return err
	}
	if s != NL {
		return errors.New("failed reading end delimiter")
	}
	return nil
}

func (d *MVBDecoder) ReadFrame(fcode *FCode) (Frame, error) {
	err := d.WaitUntilStartOfFrame()
	if err != nil {
		return nil, err
	}
	isMaster, err := d.ReadStartDelimiter()
	if err != nil {
		return nil, err
	}
	if isMaster {
		return d.ReadMaster()
	} else {
		if fcode == nil {
			return nil, fmt.Errorf("unexpected slave frame")
		}
		return d.ReadSlave(fcode)
	}
}

func (d *MVBDecoder) ReadMaster() (Frame, error) {
	// 3.4.1.1 Master Frame format
	// 3.5.2.1 Master Frame format

	var data [2]byte
	err := d.ReadBytes(data[:])
	if err != nil {
		return nil, err
	}

	cs, err := d.ReadByte()
	if err != nil {
		return nil, err
	}
	err = checkCRC(data[:], cs)
	if err != nil {
		return nil, err
	}

	fcode := data[0] >> 4
	address := ((uint16(data[0]) & 0xf) << 8) | uint16(data[1])
	err = d.ReadEndDelimiter()
	if err != nil {
		return nil, err
	}
	return &MasterFrame{fcode, address}, nil
}

func (d *MVBDecoder) ReadSlave(fcode *FCode) (Frame, error) {
	data := make([]byte, fcode.SlaveFrameSize/8)
	i := 0
	for i < len(data) {
		// one check sequence every 8 bytes
		chunk := data[i:]
		if len(chunk) > 8 {
			chunk = data[i : i+8]
		}
		err := d.ReadBytes(chunk)
		if err != nil {
			return nil, err
		}

		cs, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		err = checkCRC(chunk, cs)
		if err != nil {
			return nil, err
		}

		i += len(chunk)
	}
	err := d.ReadEndDelimiter()
	if err != nil {
		return nil, err
	}
	return &SlaveFrame{data}, nil
}

func (d *MVBDecoder) Loop(events chan<- Event) {
	var master *MasterFrame
	for {
		var frame Frame
		var fcode *FCode
		var err error

		_, err = d.stream.WaitUntilIdle(BT_SAMPLES * 2)
		if err != nil {
			goto onError
		}
		if master != nil {
			fcode = fcodes[master.FCode]
		}
		frame, err = d.ReadFrame(fcode)
		if err != nil {
			goto onError
		}
		if frame.IsMaster() {
			if master != nil {
				events <- &Telegram{n: d.stream.N(), Master: master}
			}
			master = frame.(*MasterFrame)
		} else {
			if master == nil {
				err = errors.New("unexpected slave frame")
				goto onError
			}
			events <- &Telegram{n: d.stream.N(), Master: master, Slave: frame.(*SlaveFrame)}
			master = nil
		}
		continue

	onError:
		events <- Error{error: err, n: d.stream.N(), samples: d.stream.GetSamples()}
	}
}
