package upgrade

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"runtime"
)

func CheckBinary(runtimeID string, body []byte) error {
	bad := errors.New("包的内容不正确")
	parts := bytes.Split([]byte(runtimeID), []byte("/"))
	if len(parts) != 3 || string(parts[0]) != "go" {
		return bad
	}
	switch string(parts[1]) {
	case "linux":
		if len(body) >= 4 && bytes.HasPrefix(body, []byte{0x7f, 'E', 'L', 'F'}) {
			return nil
		}
	case "darwin":
		if len(body) >= 4 && isMachO(body[:4]) {
			return nil
		}
	case "windows":
		if len(body) >= 2 && bytes.HasPrefix(body, []byte{'M', 'Z'}) {
			return nil
		}
	}
	return bad
}

func isMachO(magic []byte) bool {
	value := binary.BigEndian.Uint32(magic)
	switch value {
	case 0xFEEDFACE, 0xFEEDFACF, 0xCAFEBABE, 0xCEFAEDFE, 0xCFFAEDFE, 0xBEBAFECA:
		return true
	default:
		return false
	}
}

var renameFn = os.Rename

func ApplyFile(program string, payload []byte) error {
	incoming := program + ".incoming"
	previous := program + ".previous"
	if err := os.WriteFile(incoming, payload, 0o755); err != nil {
		_ = os.Remove(incoming)
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(incoming, 0o755); err != nil {
			_ = os.Remove(incoming)
			return err
		}
	}
	if err := os.Remove(previous); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(incoming)
		return err
	}
	if err := renameFn(program, previous); err != nil {
		_ = os.Remove(incoming)
		return err
	}
	if err := renameFn(incoming, program); err != nil {
		_ = renameFn(previous, program)
		_ = os.Remove(incoming)
		return err
	}
	return nil
}
