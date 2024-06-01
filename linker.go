package png2prg

import (
	"fmt"
	"io"
	"log"
	"os"
)

type Word uint16

func NewWord(bLo, bHi byte) Word {
	return Word(uint16(bHi)<<8 + uint16(bLo))
}
func (w Word) String() string {
	return fmt.Sprintf("%#04x", uint16(w))
}
func (w Word) Low() byte {
	return byte(w & 0xff)
}
func (w Word) High() byte {
	return byte(w >> 8)
}
func (w Word) Bytes() []byte {
	return []byte{w.Low(), w.High()}
}

const MaxMemory = 0xffff

type Linker struct {
	Verbose bool
	cursor  Word
	payload [MaxMemory + 1]byte
	block   [MaxMemory + 1]bool
	used    [MaxMemory + 1]bool
}

// NewLinker returns an empty linker with cursor set to start. When verbose is true, WriteTo also writes the memory map to os.Stdout.
func NewLinker(start Word, verbose bool) *Linker {
	return &Linker{cursor: start, Verbose: verbose}
}

// Used returns true if the current byte is already used.
func (l *Linker) Used() bool {
	return l.used[l.cursor] || l.block[l.cursor]
}

// Block blocks the memory area from start to end that must be kept free.
// Blocked memory is not included in output, unless inbetween l.used bytes.
func (l *Linker) Block(start, end Word) {
	for i := start; i < end; i++ {
		l.block[i] = true
	}
}

// Cursor returns the current cursor or memory address where the next Write will be stored.
func (l *Linker) Cursor() Word {
	return l.cursor
}

// SetCursor sets the current memory address where the next Write will be stored.
func (l *Linker) SetCursor(v Word) {
	l.cursor = v
}

// SetByte writes v at addr, regardless if it's in use or not. Useful for patching bytes.
func (l *Linker) SetByte(addr Word, v ...byte) {
	for i := range v {
		l.payload[int(addr)+int(i)] = v[i]
		l.used[int(addr)+int(i)] = true
	}
}

// CursorWrite sets the cursor and writes b to payload.
func (l *Linker) CursorWrite(cursor Word, b []byte) (n int, err error) {
	l.cursor = cursor
	return l.Write(b)
}

type LinkMap map[Word][]byte

// MapWrite writes all byteslices to the linker at their respective addresses.
func (l *Linker) WriteMap(m LinkMap) (n int, err error) {
	for c, bin := range m {
		p, err := l.CursorWrite(c, bin)
		n += p
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Write writes b to payload at cursor address and increases the cursor with amount of bytes written.
func (l *Linker) Write(b []byte) (n int, err error) {
	if int(l.cursor)+len(b) > MaxMemory {
		return n, fmt.Errorf("linker.Write: out of memory error, cursor %s, length %#04x", l.cursor, len(b))
	}
	for i := 0; i < len(b); i++ {
		if l.Used() {
			if n, err = l.WriteMemoryUsage(os.Stdout); err != nil {
				log.Printf("l.WriteMemoryUsage failed: %v", err)
			}
			return n, fmt.Errorf("linker.Write: memory overlap error, cursor %s, length %#04x", l.cursor, len(b)-i)
		}
		l.payload[l.cursor] = b[i]
		l.used[l.cursor] = true
		l.cursor++
		n++
	}
	return n, nil
}

// WritePrg writes the contents of the prg to the startaddress (first 2 bytes) in the Linker.
func (l *Linker) WritePrg(prg []byte) (n int, err error) {
	if len(prg) < 3 {
		return 0, fmt.Errorf("prg too short to write. length: %d", len(prg))
	}
	l.cursor = NewWord(prg[0], prg[1])
	return l.Write(prg[2:])
}

// StartAddress returns the memory address of the first used byte.
func (l *Linker) StartAddress() Word {
	for i := 0; i <= MaxMemory; i++ {
		if l.used[i] {
			return Word(i)
		}
	}
	return MaxMemory
}

// EndAddress returns the memory address of the last used byte + 1.
func (l *Linker) EndAddress() Word {
	for i := MaxMemory; i >= 0; i-- {
		if l.used[i] {
			return Word(i + 1)
		}
	}
	return 0
}

// LastAddress returns the memory address of the last used or blocked byte + 1.
func (l *Linker) LastAddress() Word {
	for i := MaxMemory; i >= 0; i-- {
		if l.used[i] || l.block[i] {
			return Word(i + 1)
		}
	}
	return 0
}

// Bytes returns a slice of the used payload.
func (l *Linker) Bytes() []byte {
	start := l.StartAddress()
	end := l.EndAddress()
	if end < start {
		return []byte{}
	}
	return l.payload[start:end]
}

// WriteTo writes the 2 byte startaddress and all used memory linked in one .prg to w.
func (l *Linker) WriteTo(w io.Writer) (n int64, err error) {
	start := l.StartAddress()
	end := l.EndAddress()
	if start >= end {
		return n, fmt.Errorf("linker: Write failed %s >= %s: %w", start, end, err)
	}
	m, err := w.Write(start.Bytes())
	n = int64(m)
	if err != nil {
		return n, fmt.Errorf("linker: Write failed start address %s: %w", start, err)
	}
	m, err = w.Write(l.payload[start:end])
	n += int64(m)
	if err != nil {
		return n, fmt.Errorf("linker: Write failed %s - %s: %w", start, end, err)
	}
	if l.Verbose {
		if _, err = l.WriteMemoryUsage(os.Stdout); err != nil {
			return n, fmt.Errorf("linker: WriteMemoryUsage failed: %w", err)
		}
	}
	return n, nil
}

// WriteMemoryUsage writes memory usage map in text form to w.
func (l *Linker) WriteMemoryUsage(w io.Writer) (n int, err error) {
	fmt.Fprintln(w, "memory usage:")
	eof := l.LastAddress()
	for k := 0; k < 16; k++ {
		s := ""
		for p := 0; p < 16; p++ {
			used := false
			blocked := false
			for i := 0; i < 0x100; i++ {
				if l.used[k*0x1000+p*0x100+i] {
					used = true
					break
				}
				if l.block[k*0x1000+p*0x100+i] {
					blocked = true
					break
				}
			}
			switch {
			case used:
				s += "+"
			case blocked:
				s += "x"
			default:
				s += "."
			}
		}
		m, err := fmt.Fprintf(w, "%#04x: %s\n", k*0x1000, s)
		n += m
		if err != nil {
			return n, err
		}
		if k*0x1000 > int(eof) {
			return n, nil
		}
	}
	return n, nil
}
