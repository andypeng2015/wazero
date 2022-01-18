package text

import (
	"os"
	"testing"
	"unicode/utf8"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/stretchr/testify/require"

	"github.com/tetratelabs/wazero/wasm"
	"github.com/tetratelabs/wazero/wasm/binary"
)

func TestTextToBinary(t *testing.T) {
	zero, four := uint32(0), uint32(4)
	f32, i32, i64 := wasm.ValueTypeF32, wasm.ValueTypeI32, wasm.ValueTypeI64
	tests := []struct {
		name     string
		input    string
		expected *wasm.Module
	}{
		{
			name:     "empty",
			input:    "(module)",
			expected: &wasm.Module{},
		},
		{
			name:  "import func empty",
			input: "(module (import \"foo\" \"bar\" (func)))", // ok empty sig
			expected: &wasm.Module{
				TypeSection: []*wasm.FunctionType{{}},
				ImportSection: []*wasm.Import{{
					Module: "foo", Name: "bar",
					Kind:     wasm.ImportKindFunc,
					DescFunc: 0,
				}},
			},
		},
		{
			name: "multiple import func with different inlined type",
			input: `(module
	(type (func) (; ensures no false match on index 0 ;))
	(import "wasi_snapshot_preview1" "path_open" (func $runtime.path_open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
	(import "wasi_snapshot_preview1" "fd_write" (func $runtime.fd_write (param i32 i32 i32 i32) (result i32)))
)`,
			expected: &wasm.Module{
				TypeSection: []*wasm.FunctionType{
					{},
					{Params: []wasm.ValueType{i32, i32, i32, i32, i32, i64, i64, i32, i32}, Results: []wasm.ValueType{i32}},
					{Params: []wasm.ValueType{i32, i32, i32, i32}, Results: []wasm.ValueType{i32}},
				},
				ImportSection: []*wasm.Import{
					{
						Module: "wasi_snapshot_preview1", Name: "path_open",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 1,
					}, {
						Module: "wasi_snapshot_preview1", Name: "fd_write",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 2,
					},
				},
				NameSection: &wasm.NameSection{
					FunctionNames: map[uint32]string{
						0: "runtime.path_open",
						1: "runtime.fd_write",
					},
				},
			},
		},
		{
			name: "multiple import func different type - name index",
			input: `(module
	(type (func) (; ensures no false match on index 0 ;))
	(type $i32i32_i32 (func (param i32 i32) (result i32)))
	(type $i32i32i32i32_i32 (func (param i32 i32 i32 i32) (result i32)))
	(import "wasi_snapshot_preview1" "arg_sizes_get" (func $runtime.arg_sizes_get (type $i32i32_i32)))
	(import "wasi_snapshot_preview1" "fd_write" (func $runtime.fd_write (type $i32i32i32i32_i32)))
)`,
			expected: &wasm.Module{
				TypeSection: []*wasm.FunctionType{
					{},
					{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32}},
					{Params: []wasm.ValueType{i32, i32, i32, i32}, Results: []wasm.ValueType{i32}},
				},
				ImportSection: []*wasm.Import{
					{
						Module: "wasi_snapshot_preview1", Name: "arg_sizes_get",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 1,
					}, {
						Module: "wasi_snapshot_preview1", Name: "fd_write",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 2,
					},
				},
				NameSection: &wasm.NameSection{
					FunctionNames: map[uint32]string{
						0: "runtime.arg_sizes_get",
						1: "runtime.fd_write",
					},
				},
			},
		},
		{
			name: "start imported function by name",
			input: `(module
	(import "" "hello" (func $hello))
	(start $hello)
)`,
			expected: &wasm.Module{
				TypeSection: []*wasm.FunctionType{{}},
				ImportSection: []*wasm.Import{{
					Module: "", Name: "hello",
					Kind:     wasm.ImportKindFunc,
					DescFunc: 0,
				}},
				StartSection: &zero,
				NameSection: &wasm.NameSection{
					FunctionNames: map[uint32]string{
						0: "hello",
					},
				},
			},
		},
		{
			name: "start imported function by index",
			input: `(module
	(import "" "hello" (func))
	(start 0)
)`,
			expected: &wasm.Module{
				TypeSection: []*wasm.FunctionType{{}},
				ImportSection: []*wasm.Import{{
					Module: "", Name: "hello",
					Kind:     wasm.ImportKindFunc,
					DescFunc: 0,
				}},
				StartSection: &zero,
			},
		},
		{
			name:  "example",
			input: string(example),
			expected: &wasm.Module{
				TypeSection: []*wasm.FunctionType{
					{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32}},
					{},
					{Params: []wasm.ValueType{i32, i32, i32, i32}, Results: []wasm.ValueType{i32}},
					{Params: []wasm.ValueType{f32, f32}, Results: []wasm.ValueType{f32}},
				},
				ImportSection: []*wasm.Import{
					{
						Module: "wasi_snapshot_preview1", Name: "arg_sizes_get",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 0,
					}, {
						Module: "wasi_snapshot_preview1", Name: "fd_write",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 2,
					}, {
						Module: "Math", Name: "Mul",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 3,
					}, {
						Module: "Math", Name: "Add",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 0,
					}, {
						Module: "", Name: "hello",
						Kind:     wasm.ImportKindFunc,
						DescFunc: 1,
					},
				},
				StartSection: &four,
				NameSection: &wasm.NameSection{
					ModuleName: "example",
					FunctionNames: map[uint32]string{
						0: "runtime.arg_sizes_get",
						1: "runtime.fd_write",
						2: "mul",
						3: "add",
						4: "hello",
					},
					LocalNames: map[uint32]map[uint32]string{
						1: {0: "fd", 1: "iovs_ptr", 2: "iovs_len", 3: "nwritten_ptr"},
						2: {0: "x", 1: "y"},
						3: {0: "l", 1: "r"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			m, err := DecodeModule([]byte(tc.input))
			require.NoError(t, err)
			require.Equal(t, tc.expected, m)
		})
	}
}

func TestTextToBinary_Errors(t *testing.T) {
	tests := []struct{ name, input, expectedErr string }{
		{
			name:        "invalid",
			input:       "module",
			expectedErr: "1:1: expected '(', but found keyword: module",
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeModule([]byte(tc.input))
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func BenchmarkTextToBinaryExample(b *testing.B) {
	var exampleBinary []byte // wat2wasm --debug-names example.wat
	if bin, err := os.ReadFile("testdata/example.wasm"); err != nil {
		b.Fatal(err)
	} else {
		exampleBinary = bin
	}

	b.Run("vs utf8.Valid", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !utf8.Valid(example) {
				panic("unexpected")
			}
		}
	})
	// Not a fair comparison as while DecodeModule parses into the binary format, we don't encode it into a byte slice.
	// We also don't know if wasmtime.Wat2Wasm encodes the custom name section or not.
	b.Run("vs wasmtime.Wat2Wasm", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := wasmtime.Wat2Wasm(string(example))
			if err != nil {
				panic(err)
			}
		}
	})
	// This compares against reading the same binary data directly (encoded via wat2wasm --debug-names).
	// Note: This will be more similar once DecodeModule writes CustomSection["name"]
	b.Run("vs binary.DecodeModule", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := binary.DecodeModule(exampleBinary); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("vs wat.lex", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if line, col, err := lex(noopTokenParser, example); err != nil {
				b.Fatalf("%d:%d: %s", line, col, err)
			}
		}
	})
	b.Run("vs wat.parseModule", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := parseModule(example); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("DecodeModule", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := DecodeModule(example); err != nil {
				b.Fatal(err)
			}
		}
	})
}