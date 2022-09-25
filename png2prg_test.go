package png2prg

import (
	"bytes"
	"testing"
)

const inFile = "testdata/floris_untitled.png"
const testSID = "testdata/Laserdance_10.sid"

func BenchmarkNewFromPath(b *testing.B) {
	opt := Options{
		Quiet:      true,
		Display:    true,
		IncludeSID: testSID,
	}
	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		p, err := NewFromPath(opt, inFile)
		if err != nil {
			b.Fatalf("NewFromPath %q failed: %v", inFile, err)
		}
		if _, err = p.WriteTo(buf); err != nil {
			b.Fatalf("WriteTo failed: %v", err)
		}
	}
}

func BenchmarkNewFromPathParallel(b *testing.B) {
	opt := Options{
		Quiet:      true,
		Display:    true,
		IncludeSID: testSID,
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := &bytes.Buffer{}
			p, err := NewFromPath(opt, inFile)
			if err != nil {
				b.Fatalf("NewFromPath %q failed: %v", inFile, err)
			}
			if _, err = p.WriteTo(buf); err != nil {
				b.Fatalf("WriteTo failed: %v", err)
			}
		}
	})
}

func BenchmarkNewFromPathNoCrunch(b *testing.B) {
	opt := Options{
		Quiet:      true,
		Display:    true,
		NoCrunch:   true,
		IncludeSID: testSID,
	}
	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		p, err := NewFromPath(opt, inFile)
		if err != nil {
			b.Fatalf("NewFromPath %q failed: %v", inFile, err)
		}
		if _, err = p.WriteTo(buf); err != nil {
			b.Fatalf("WriteTo failed: %v", err)
		}
	}
}

func BenchmarkNewFromPathNoCrunchParallel(b *testing.B) {
	opt := Options{
		Quiet:      true,
		Display:    true,
		NoCrunch:   true,
		IncludeSID: testSID,
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := &bytes.Buffer{}
			p, err := NewFromPath(opt, inFile)
			if err != nil {
				b.Fatalf("NewFromPath %q failed: %v", inFile, err)
			}
			if _, err = p.WriteTo(buf); err != nil {
				b.Fatalf("WriteTo failed: %v", err)
			}
		}
	})
}
