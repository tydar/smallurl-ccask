package ccask

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// TODO: try to think about a reasonable response size limit
//       with an eye toward the protocol as defined in ccask
// TODO: use context.Context?
// Maybe TODO: look into github.com/jackc/puddle for pooling

var ErrBadCommand = errors.New("Unsupported command code")
var ErrMsgTooBig = errors.New("Message too big")
var ErrMsgIncomplete = errors.New("Message incomplete")
var ErrUnknownResponseCode = errors.New("Message has unknown response code")
var ErrBufOverflow = errors.New("Provided buffer too small")

const CMD_HEADER_SZ = 13
const RES_HEADER_SZ = 9
const MAX_RES_SIZE = 4096 // 4KB response size limit, arbitrary

type CCaskClient struct {
	Port       string
	Domain     string
	MaxMsgSize uint32

	conn          net.Conn
	cmdMarshaller func(CCaskCmdMsg) ([]byte, error)
	mu            sync.Mutex
}

func NewCCaskClient(port string, domain string, maxMsgSize uint32) *CCaskClient {
	return &CCaskClient{
		Port:          port,
		Domain:        domain,
		MaxMsgSize:    maxMsgSize,
		conn:          nil,
		cmdMarshaller: nil,
	}
}

func (cc *CCaskClient) Connect() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.cmdMarshaller = cc.CmdMarshallerFactory()

	conn, err := net.Dial("tcp", cc.Domain+":"+cc.Port)
	if err != nil {
		return fmt.Errorf("net.Dial: %w", err)
	}

	cc.conn = conn
	return nil
}

func (cc *CCaskClient) Disconnect() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if err := cc.conn.Close(); err != nil {
		return fmt.Errorf("Close: %w", err)
	}

	return nil
}

func (cc *CCaskClient) GetRes(key []byte) (CCaskResponse, error) {
	buf, err := cc.Get(key)
	if err != nil {
		return CCaskResponse{}, fmt.Errorf("Get: %w", err)
	}

	return UnmarshalCCaskResponse(buf)
}

func (cc *CCaskClient) Get(key []byte) ([]byte, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cmd := NewCCaskCmdMsg(GET, key, []byte{})
	resultBytes := make([]byte, MAX_RES_SIZE)

	cmd_bytes, err := cc.cmdMarshaller(cmd)
	if err != nil {
		return resultBytes, fmt.Errorf("cmdMarshaller: %w", err)
	}

	written := 0

	// try to write bytes
	n, err := cc.conn.Write(cmd_bytes)
	if err != nil {
		return resultBytes, fmt.Errorf("conn.Write: %w", err)
	}

	// write until full cmd sent
	written += n
	for written < len(cmd_bytes) {
		n, err = cc.conn.Write(cmd_bytes[written:])
		if err != nil {
			return resultBytes, fmt.Errorf("conn.Write: %w", err)
		}

		written += n
	}

	if err := cc.receiveResponse(resultBytes); err != nil {
		return []byte{}, fmt.Errorf("receiveResponse: %w", err)
	}

	return resultBytes, nil
}

func (cc *CCaskClient) SetRes(key, value []byte) (CCaskResponse, error) {
	buf, err := cc.Set(key, value)
	if err != nil {
		return CCaskResponse{}, fmt.Errorf("Set: %w", err)
	}

	return UnmarshalCCaskResponse(buf)
}

func (cc *CCaskClient) Set(key []byte, value []byte) ([]byte, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cmd := NewCCaskCmdMsg(SET, key, value)
	resultBytes := make([]byte, 1024) // smaller max size since we get short fixed messages here

	cmdBytes, err := cc.cmdMarshaller(cmd)
	if err != nil {
		return resultBytes, fmt.Errorf("cmdMarshaller: %w", err)
	}

	written := 0
	for n, err := cc.conn.Write(cmdBytes[written:]); written < len(cmdBytes); written += n {
		if err != nil {
			return resultBytes, fmt.Errorf("Write: %w", err)
		}

	}

	if err := cc.receiveResponse(resultBytes); err != nil {
		return []byte{}, fmt.Errorf("receiveResponse %w", err)
	}

	return resultBytes, nil
}

func (cc *CCaskClient) receiveResponse(b []byte) error {
	buf := make([]byte, 4)
	n, err := io.ReadFull(cc.conn, buf)
	if err != nil {
		return fmt.Errorf("ReadFull: %w", err)
	}

	if n < 4 {
		// didn't get a full msglen
		return fmt.Errorf("%w", ErrMsgIncomplete)
	}

	msglen := binary.BigEndian.Uint32(buf)

	msgbuf := make([]byte, msglen-4)
	n, err = io.ReadFull(cc.conn, msgbuf)
	if err != nil {
		return fmt.Errorf("ReadFull: %w", err)
	}

	if uint32(n+4) < msglen {
		return fmt.Errorf("%w", ErrMsgIncomplete)
	}

	if uint32(len(b)) < 4+msglen {
		return fmt.Errorf("%w", ErrBufOverflow)
	}

	copy(b, buf)
	copy(b[4:], msgbuf)

	return nil
}

func (cc *CCaskClient) CmdMarshallerFactory() func(CCaskCmdMsg) ([]byte, error) {
	return func(cm CCaskCmdMsg) ([]byte, error) {
		if cm.cmdCode != GET && cm.cmdCode != SET {
			return []byte{}, fmt.Errorf("%d: %w", cm.cmdCode, ErrBadCommand)
		}

		keySz := uint32(len(cm.key))
		valSz := uint32(len(cm.value))
		maxSizeMinusHeader := cc.MaxMsgSize - CMD_HEADER_SZ
		if maxSizeMinusHeader-keySz < valSz || maxSizeMinusHeader-valSz < keySz ||
			keySz+valSz+CMD_HEADER_SZ > cc.MaxMsgSize {
			return []byte{}, fmt.Errorf("%w", ErrMsgTooBig)
		}

		acc := make([]byte, CMD_HEADER_SZ+keySz+valSz)
		index := 0

		binary.BigEndian.PutUint32(acc[index:index+4], CMD_HEADER_SZ+keySz+valSz)
		index += 4

		acc[index] = byte(cm.cmdCode)
		index += 1

		binary.BigEndian.PutUint32(acc[index:index+4], keySz)
		index += 4

		binary.BigEndian.PutUint32(acc[index:index+4], valSz)
		index += 4

		copy(acc[index:index+int(keySz)], cm.key)
		index += int(keySz)

		copy(acc[index:index+int(valSz)], cm.value)

		return acc, nil
	}
}

type CCaskCmdCode byte

const (
	GET CCaskCmdCode = iota
	SET
)

type CCaskCmdMsg struct {
	cmdCode CCaskCmdCode
	key     []byte
	value   []byte
}

func NewCCaskCmdMsg(cmdCode CCaskCmdCode, key, value []byte) CCaskCmdMsg {
	return CCaskCmdMsg{
		cmdCode: cmdCode,
		key:     key,
		value:   value,
	}
}

func (cc CCaskCmdMsg) CmdCode() CCaskCmdCode {
	return cc.cmdCode
}

func (cc CCaskCmdMsg) Key() []byte {
	return cc.key
}

func (cc CCaskCmdMsg) Value() []byte {
	return cc.value
}

type CCaskResCode byte

const (
	GET_SUCCESS CCaskResCode = iota
	GET_FAIL
	SET_SUCCESS
	SET_FAIL
	BAD_COMMAND
)

type CCaskResponse struct {
	resCode CCaskResCode
	value   []byte
}

func (cr CCaskResponse) ResCode() CCaskResCode {
	return cr.resCode
}

func (cr CCaskResponse) Value() []byte {
	return cr.value
}

func UnmarshalCCaskResponse(data []byte) (CCaskResponse, error) {
	sz := len(data)
	if sz < RES_HEADER_SZ {
		return CCaskResponse{}, fmt.Errorf("%w", ErrMsgIncomplete)
	}

	msglen := binary.BigEndian.Uint32(data[0:4])
	if msglen == 0 || msglen > uint32(sz) {
		return CCaskResponse{}, fmt.Errorf("%w", ErrMsgIncomplete)
	}
	index := 4

	truncData := data[:msglen]
	resCode := CCaskResCode(truncData[index])
	index += 1

	if resCode != GET_SUCCESS && resCode != GET_FAIL && resCode != SET_SUCCESS &&
		resCode != SET_FAIL && resCode != BAD_COMMAND {
		return CCaskResponse{}, fmt.Errorf("%w", ErrUnknownResponseCode)
	}

	vsz := binary.BigEndian.Uint32(truncData[index : index+4])
	index += 4

	value := truncData[index : index+int(vsz)]

	return CCaskResponse{
		resCode: resCode,
		value:   value,
	}, nil
}
