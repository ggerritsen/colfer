package colfer

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Generate writes the code into file "Colfer.go".
func Generate(basedir string, packages []*Package) error {
	t := template.New("go-code").Delims("<:", ":>")
	template.Must(t.Parse(goCode))
	template.Must(t.New("marshal-field").Parse(goMarshalField))
	template.Must(t.New("marshal-field-len").Parse(goMarshalFieldLen))
	template.Must(t.New("marshal-varint").Parse(goMarshalVarint))
	template.Must(t.New("marshal-varint-len").Parse(goMarshalVarintLen))
	template.Must(t.New("unmarshal-field").Parse(goUnmarshalField))
	template.Must(t.New("unmarshal-header").Parse(goUnmarshalHeader))
	template.Must(t.New("unmarshal-varint32").Parse(goUnmarshalVarint32))
	template.Must(t.New("unmarshal-varint64").Parse(goUnmarshalVarint64))

	for _, p := range packages {
		p.NameNative = p.Name[strings.LastIndexByte(p.Name, '/')+1:]
	}

	for _, p := range packages {
		for _, s := range p.Structs {
			for _, f := range s.Fields {
				switch f.Type {
				default:
					if f.TypeRef == nil {
						f.TypeNative = f.Type
					} else {
						f.TypeNative = f.TypeRef.NameTitle()
						if f.TypeRef.Pkg != p {
							f.TypeNative = f.TypeRef.Pkg.NameNative + "." + f.TypeNative
						}
					}
				case "timestamp":
					f.TypeNative = "time.Time"
				case "text":
					f.TypeNative = "string"
				case "binary":
					f.TypeNative = "[]byte"
				}
			}
		}

		pkgdir, err := makePkgDir(p, basedir)
		if err != nil {
			return err
		}
		f, err := os.Create(filepath.Join(pkgdir, "Colfer.go"))
		if err != nil {
			return err
		}
		defer f.Close()

		if err = t.Execute(f, p); err != nil {
			return err
		}
	}
	return nil
}

func makePkgDir(p *Package, basedir string) (path string, err error) {
	pkgdir := strings.Replace(p.Name, "/", string(filepath.Separator), -1)
	path = filepath.Join(basedir, pkgdir)
	err = os.MkdirAll(path, os.ModeDir|os.ModePerm)
	return
}

const goCode = `package <:.NameNative:>

// This file was generated by colf(1); DO NOT EDIT

import (
	"fmt"
	"io"
	"math"
	"time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = math.E
var _ = time.RFC3339

// ColferContinue signals a data continuation as a byte index.
type ColferContinue int

func (i ColferContinue) Error() string {
	return fmt.Sprintf("colfer: data continuation at byte %d", i)
}

// ColferError signals a data mismatch as as a byte index.
type ColferError int

func (i ColferError) Error() string {
	return fmt.Sprintf("colfer: unknown header at byte %d", i)
}
<:range .Structs:>
type <:.NameTitle:> struct {
<:range .Fields:>	<:.NameTitle:>	<:if .TypeArray:>[]<:end:><:if .TypeRef:>*<:end:><:.TypeNative:>
<:end:>}

// MarshalTo encodes o as Colfer into buf and returns the number of bytes written.
// If the buffer is too small, MarshalTo will panic.
func (o *<:.NameTitle:>) MarshalTo(buf []byte) int {
	if o == nil {
		return 0
	}

	var i int
<:range .Fields:><:template "marshal-field" .:><:end:>
	buf[i] = 0x7f
	i++
	return i
}

// MarshalLen returns the Colfer serial byte size.
func (o *<:.NameTitle:>) MarshalLen() int {
	if o == nil {
		return 0
	}

	l := 1
<:range .Fields:><:template "marshal-field-len" .:><:end:>
	return l
}

// MarshalBinary encodes o as Colfer conform encoding.BinaryMarshaler.
// The error return is always nil.
func (o *<:.NameTitle:>) MarshalBinary() (data []byte, err error) {
	data = make([]byte, o.MarshalLen())
	o.MarshalTo(data)
	return data, nil
}

// UnmarshalBinary decodes data as Colfer conform encoding.BinaryUnmarshaler.
// The error return options are io.EOF, <:.Pkg.Name:>.ColferError, and <:.Pkg.Name:>.ColferContinue.
func (o *<:.NameTitle:>) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return io.EOF
	}

	header := data[0]
	i := 1
<:range .Fields:><:template "unmarshal-field" .:><:end:>
	if header != 0x7f {
		return ColferError(i - 1)
	}
	if i != len(data) {
		return ColferContinue(i)
	}
	return nil
}
<:end:>`

const goMarshalField = `<:if eq .Type "bool":>
	if o.<:.NameTitle:> {
		buf[i] = <:.Index:>
		i++
	}
<:else if eq .Type "uint32":>
	if x := o.<:.NameTitle:>; x != 0 {
		buf[i] = <:.Index:>
		i++
<:template "marshal-varint":>
	}
<:else if eq .Type "uint64":>
	if x := o.<:.NameTitle:>; x != 0 {
		buf[i] = <:.Index:>
		i++
<:template "marshal-varint":>
	}
<:else if eq .Type "int32":>
	if v := o.<:.NameTitle:>; v != 0 {
		x := uint32(v)
		if v >= 0 {
			buf[i] = <:.Index:>
		} else {
			x = ^x + 1
			buf[i] = <:.Index:> | 0x80
		}
		i++
<:template "marshal-varint":>
	}
<:else if eq .Type "int64":>
	if v := o.<:.NameTitle:>; v != 0 {
		x := uint64(v)
		if v >= 0 {
			buf[i] = <:.Index:>
		} else {
			x = ^x + 1
			buf[i] = <:.Index:> | 0x80
		}
		i++
<:template "marshal-varint":>
	}
<:else if eq .Type "float32":>
	if v := o.<:.NameTitle:>; v != 0.0 {
		buf[i] = <:.Index:>
		x := math.Float32bits(v)
		buf[i+1], buf[i+2], buf[i+3], buf[i+4] = byte(x>>24), byte(x>>16), byte(x>>8), byte(x)
		i += 5
	}
<:else if eq .Type "float64":>
	if v := o.<:.NameTitle:>; v != 0.0 {
		buf[i] = <:.Index:>
		x := math.Float64bits(v)
		buf[i+1], buf[i+2], buf[i+3], buf[i+4] = byte(x>>56), byte(x>>48), byte(x>>40), byte(x>>32)
		buf[i+5], buf[i+6], buf[i+7], buf[i+8] = byte(x>>24), byte(x>>16), byte(x>>8), byte(x)
		i += 9
	}
<:else if eq .Type "timestamp":>
	if v := o.<:.NameTitle:>; !v.IsZero() {
		buf[i] = <:.Index:>
		s, ns := v.Unix(), v.Nanosecond()
		buf[i+1], buf[i+2], buf[i+3], buf[i+4] = byte(s>>56), byte(s>>48), byte(s>>40), byte(s>>32)
		buf[i+5], buf[i+6], buf[i+7], buf[i+8] = byte(s>>24), byte(s>>16), byte(s>>8), byte(s)
		if ns == 0 {
			i += 9
		} else {
			buf[i] |= 0x80
			buf[i+9], buf[i+10], buf[i+11], buf[i+12] = byte(ns>>24), byte(ns>>16), byte(ns>>8), byte(ns)
			i += 13
		}
	}
<:else if eq .Type "text" "binary":>
	if v := o.<:.NameTitle:>; len(v) != 0 {
		buf[i] = <:.Index:>
		i++
		x := uint(len(v))
<:template "marshal-varint":>
		copy(buf[i:], v)
		i += len(v)
	}
<:else if .TypeArray:>
	if l := len(o.<:.NameTitle:>); l != 0 {
		buf[i] = <:.Index:>
		i++
		x := uint(l)
<:template "marshal-varint":>
		for _, v := range o.<:.NameTitle:> {
			i += v.MarshalTo(buf[i:])
		}
	}
<:else:>
	if v := o.<:.NameTitle:>; v != nil {
		buf[i] = <:.Index:>
		i++
		i += v.MarshalTo(buf[i:])
	}
<:end:>`

const goMarshalFieldLen = `<:if eq .Type "bool":>
	if o.<:.NameTitle:> {
		l++
	}
<:else if eq .Type "uint32":>
	if x := o.<:.NameTitle:>; x != 0 {
<:template "marshal-varint-len" .:>
	}
<:else if eq .Type "uint64":>
	if x := o.<:.NameTitle:>; x != 0 {
<:template "marshal-varint-len" .:>
	}
<:else if eq .Type "int32":>
	if v := o.<:.NameTitle:>; v != 0 {
		x := uint32(v)
		if v < 0 {
			x = ^x + 1
		}
<:template "marshal-varint-len" .:>
	}
<:else if eq .Type "int64":>
	if v := o.<:.NameTitle:>; v != 0 {
		x := uint64(v)
		if v < 0 {
			x = ^x + 1
		}
<:template "marshal-varint-len" .:>
	}
<:else if eq .Type "float32":>
	if o.<:.NameTitle:> != 0.0 {
		l += 5
	}
<:else if eq .Type "float64":>
	if o.<:.NameTitle:> != 0.0 {
		l += 9
	}
<:else if eq .Type "timestamp":>
	if v := o.<:.NameTitle:>; !v.IsZero() {
		if v.Nanosecond() == 0 {
			l += 9
		} else {
			l += 13
		}
	}
<:else if eq .Type "text" "binary":>
	if x := len(o.<:.NameTitle:>); x != 0 {
		l += x
<:template "marshal-varint-len" .:>
	}
<:else if .TypeArray:>
	if x := len(o.<:.NameTitle:>); x != 0 {
<:template "marshal-varint-len" .:>
		for _, v := range o.<:.NameTitle:> {
			l += v.MarshalLen()
		}
	}
<:else:>
	if v := o.<:.NameTitle:>; v != nil {
		l += v.MarshalLen() + 1
	}
<:end:>`

const goMarshalVarint = `		for x >= 0x80 {
			buf[i] = byte(x | 0x80)
			x >>= 7
			i++
		}
		buf[i] = byte(x)
		i++`

const goMarshalVarintLen = `		for x >= 0x80 {
			x >>= 7
			l++
		}
		l += 2`

const goUnmarshalField = `<:if eq .Type "bool":>
	if header == <:.Index:> {
		o.<:.NameTitle:> = true
<:template "unmarshal-header":>
	}
<:else if eq .Type "uint32":>
	if header == <:.Index:> {
<:template "unmarshal-varint32":>
		o.<:.NameTitle:> = x
<:template "unmarshal-header":>
	}
<:else if eq .Type "uint64":>
	if header == <:.Index:> {
<:template "unmarshal-varint64":>
		o.<:.NameTitle:> = x
<:template "unmarshal-header":>
	}
<:else if eq .Type "int32":>
	if header == <:.Index:> || header == <:.Index:>|0x80 {
<:template "unmarshal-varint32":>
		if header&0x80 != 0 {
			x = ^x + 1
		}
		o.<:.NameTitle:> = int32(x)
<:template "unmarshal-header":>
	}
<:else if eq .Type "int64":>
	if header == <:.Index:> || header == <:.Index:>|0x80 {
<:template "unmarshal-varint64":>
		if header&0x80 != 0 {
			x = ^x + 1
		}
		o.<:.NameTitle:> = int64(x)
<:template "unmarshal-header":>
	}
<:else if eq .Type "float32":>
	if header == <:.Index:> {
		if i+4 >= len(data) {
			return io.EOF
		}
		x := uint32(data[i])<<24 | uint32(data[i+1])<<16 | uint32(data[i+2])<<8 | uint32(data[i+3])
		o.<:.NameTitle:> = math.Float32frombits(x)

		header = data[i+4]
		i += 5
	}
<:else if eq .Type "float64":>
	if header == <:.Index:> {
		if i+8 >= len(data) {
			return io.EOF
		}
		x := uint64(data[i])<<56 | uint64(data[i+1])<<48 | uint64(data[i+2])<<40 | uint64(data[i+3])<<32
		x |= uint64(data[i+4])<<24 | uint64(data[i+5])<<16 | uint64(data[i+6])<<8 | uint64(data[i+7])
		o.<:.NameTitle:> = math.Float64frombits(x)

		header = data[i+8]
		i += 9
	}
<:else if eq .Type "timestamp":>
	if header == <:.Index:> {
		if i+8 >= len(data) {
			return io.EOF
		}
		sec := uint64(data[i])<<56 | uint64(data[i+1])<<48 | uint64(data[i+2])<<40 | uint64(data[i+3])<<32
		sec |= uint64(data[i+4])<<24 | uint64(data[i+5])<<16 | uint64(data[i+6])<<8 | uint64(data[i+7])
		o.<:.NameTitle:> = time.Unix(int64(sec), 0)

		header = data[i+8]
		i += 9
	} else if header == <:.Index:>|0x80 {
		if i+12 >= len(data) {
			return io.EOF
		}
		sec := uint64(data[i])<<56 | uint64(data[i+1])<<48 | uint64(data[i+2])<<40 | uint64(data[i+3])<<32
		sec |= uint64(data[i+4])<<24 | uint64(data[i+5])<<16 | uint64(data[i+6])<<8 | uint64(data[i+7])
		nsec := int64(uint(data[i+8])<<24 | uint(data[i+9])<<16 | uint(data[i+10])<<8 | uint(data[i+11]))
		o.<:.NameTitle:> = time.Unix(int64(sec), nsec)

		header = data[i+12]
		i += 13
	}
<:else if eq .Type "text":>
	if header == <:.Index:> {
<:template "unmarshal-varint32":>

		to := i + int(x)
		if to >= len(data) {
			return io.EOF
		}
		o.<:.NameTitle:> = string(data[i:to])

		header = data[to]
		i = to + 1
	}
<:else if eq .Type "binary":>
	if header == <:.Index:> {
<:template "unmarshal-varint32":>

		length := int(x)
		to := i + length
		if to >= len(data) {
			return io.EOF
		}
		v := make([]byte, length)
		copy(v, data[i:])
		o.<:.NameTitle:> = v

		header = data[to]
		i = to + 1
	}
<:else if .TypeArray:>
	if header == <:.Index:> {
<:template "unmarshal-varint32":>

		a := make([]*<:.TypeNative:>, int(x))
		for ai, _ := range a {
			v := new(<:.TypeNative:>)
			a[ai] = v

			err := v.UnmarshalBinary(data[i:])
			switch e := err.(type) {
			case ColferContinue:
				i += int(e)
			case nil:
				return io.EOF
			default:
				return err
			}
		}
		o.<:.NameTitle:> = a

		if i == len(data) {
			return io.EOF
		}
		header = data[i]
		i++
	}
<:else:>
	if header == <:.Index:> {
		v := new(<:.TypeNative:>)
		err := v.UnmarshalBinary(data[i:])
		switch e := err.(type) {
		case ColferContinue:
			i += int(e)
		case nil:
			return io.EOF
		default:
			return err
		}
		o.<:.NameTitle:> = v

		header = data[i]
		i++
	}
<:end:>`

const goUnmarshalHeader = `
		if i == len(data) {
			return io.EOF
		}
		header = data[i]
		i++`

const goUnmarshalVarint32 = `		var x uint32
		for shift := uint(0); ; shift += 7 {
			if i == len(data) {
				return io.EOF
			}
			b := data[i]
			i++
			if shift == 28 {
				x |= uint32(b) << 28
				break
			}
			x |= (uint32(b) & 0x7f) << shift
			if b < 0x80 {
				break
			}
		}`

const goUnmarshalVarint64 = `		var x uint64
		for shift := uint(0); ; shift += 7 {
			if i == len(data) {
				return io.EOF
			}
			b := data[i]
			i++
			if shift == 63 {
				x |= 1 << 63
				break
			}
			x |= (uint64(b) & 0x7f) << shift
			if b < 0x80 {
				break
			}
		}`
