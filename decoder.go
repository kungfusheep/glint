package glint

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Decoder handles type-safe decoding of type T
type Decoder[T any] struct {
	impl *decoderImpl
}

// NewDecoder constructs a decoder specialized for type T with default limits
func NewDecoder[T any]() *Decoder[T] {
	return NewDecoderWithLimits[T](DefaultLimits)
}

// NewDecoderWithLimits constructs a decoder with custom bounds checking limits
func NewDecoderWithLimits[T any](limits DecodeLimits) *Decoder[T] {
	var zero T
	impl := newDecoderWithLimits(zero, limits)
	return &Decoder[T]{impl: impl}
}

// NewDecoderUsingTag is primarily for internal use.
// Like NewDecoder but accepts a custom struct tag name for framework integration (e.g. "rpc").
func NewDecoderUsingTag[T any](usingTagName string) *Decoder[T] {
	var zero T
	impl := newDecoderUsingTag(zero, usingTagName)
	return &Decoder[T]{impl: impl}
}

// Unmarshal extracts data from bytes into a value of type T
func (d *Decoder[T]) Unmarshal(bytes []byte, v *T) error {
	return d.impl.Unmarshal(bytes, v)
}

// UnmarshalWithContext performs decoding using the supplied context
func (d *Decoder[T]) UnmarshalWithContext(bytes []byte, v *T, context DecoderContext) error {
	return d.impl.UnmarshalWithContext(bytes, v, context)
}

const smallKeys = 9 // character limit for small keys to use trie lookups

// dtrienode represents a node in the decode instruction trie
type dtrienode struct {
	children [256]*dtrienode
	field    []decodeInstruction
	word     bool // marks end of a complete word
}

// trienode forms nodes within the field lookup trie
type trienode struct {
	field    decodeInstruction
	children [128]*trienode
	word     bool // marks end of a complete word
}

// trie provides fast lookups for small field names.
// Performance degrades beyond a certain size - `smallThresh`
// defines when to switch to map-based lookups
type trie struct {
	root trienode
}

// Add inserts a field instruction at the given name path
func (t *trie) Add(name string, field decodeInstruction) {

	node := &t.root

	for i := 0; i < len(name); i++ {
		char := name[i]

		if node.children[char] == nil {
			node.children[char] = &trienode{}
		}
		node = node.children[char]
	}

	node.field = field
	node.word = true
}

// Get performs the lookup on the supplied name
func (t *trie) Get(name string) (decodeInstruction, bool) {
	node := &t.root
	var i int
start:
	if node.children[name[i]] == nil {
		return decodeInstruction{}, false
	}
	node = node.children[name[i]]

	if i++; i < len(name) {

		goto start // this allows this function to be inlined
	}
	return node.field, node.word
}

// DecodeInstructionLookup is the data structre we use for doing lookups of small names.
// there is a tipping point where this becomes less effective than a
// simple string map - see `smallThresh` for that threshold
type DecodeInstructionLookup struct {
	mu    sync.RWMutex
	Added func(hash uint, contextID uint) // a callback to be told when an item has been added to the lookup along with a context ID
	root  dtrienode
}

// add appends the supplied field into the trie against name
func (t *DecodeInstructionLookup) add(hash []byte, field []decodeInstruction, id uint) {
	t.mu.Lock()
	defer t.mu.Unlock()

	node := &t.root

	for i := 0; i < len(hash); i++ {
		char := hash[i]

		if node.children[char] == nil {
			node.children[char] = &dtrienode{}
		}
		node = node.children[char]
	}

	node.field = field
	node.word = true

	if t.Added != nil {
		_, ok := t.get(hash) // double check we can actually pull something back out before we signal added
		if ok {
			t.Added(uint(binary.LittleEndian.Uint32(hash)), id)
		}
	}
}

// get performs the lookup on the supplied hash
func (t *DecodeInstructionLookup) get(hash []byte) ([]decodeInstruction, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	node := &t.root
	var i int
start:
	if node.children[hash[i]] == nil {
		return []decodeInstruction{}, false
	}
	node = node.children[hash[i]]

	if i++; i < len(hash) {

		goto start // this allows this function to be inlined
	}
	return node.field, node.word
}

// decoder defines methods required by all decoder types during recursive decoding
type decoder interface {
	Unmarshal([]byte, any) error
	unmarshal(Reader, []decodeInstruction, any) Reader
	parseSchema(Reader, []decodeInstruction) ([]decodeInstruction, Reader, error)
	setWireType(WireType)
}

// decoderImpl holds the internal decoding state - always construct via `newDecoder`
type decoderImpl struct {
	trie     trie                         // optimized lookups for short field names
	lookup   map[string]decodeInstruction // map-based lookups for longer names (more consistent performance)
	instr    []decodeInstruction          // fixed instruction set for specialized decoders (e.g. map values)
	numfield int                          // total fields registered in lookups
	wireType WireType                     // enables runtime type validation
	lastHash uint32                       // most recent schema hash encountered
	limits   DecodeLimits                 // bounds checking configuration
	cache    DecodeInstructionLookup      // per-decoder instance cache

}

// setWireType updates the decoder's wire type from schema information
func (d *decoderImpl) setWireType(wt WireType) {
	d.wireType = wt
}

// newDecoder builds a decoder using type information extracted from a blank struct instance,
// selecting fields marked with "glint" tags.
//
// Create only ONE decoder per type - the decoder is safe for concurrent use.
// The blank struct's type must exactly match what you'll pass to Unmarshal.
// e.g mystructDecoder := glint.newDecoder(mystruct{})
func newDecoder(t any) *decoderImpl {
	return newDecoderWithLimits(t, DefaultLimits)
}

func newDecoderWithLimits(t any, limits DecodeLimits) *decoderImpl {
	return newDecoderUsingTagWithLimits(t, "glint", limits)
}

// NewDecoderUsingTag should almost never be used directly.
//
// Provides the same functionality as NewDecoder, but allows a struct tag to be customized for framework purposes (i.e. "rpc").
func newDecoderUsingTag(t any, usingTagName string) *decoderImpl {
	return newDecoderUsingTagWithLimits(t, usingTagName, DefaultLimits)
}

func newDecoderUsingTagWithLimits(t any, usingTagName string, limits DecodeLimits) *decoderImpl {
	d := &decoderImpl{}
	d.lookup = make(map[string]decodeInstruction)
	d.limits = limits

	tt := reflect.TypeOf(t)

	if tt.Kind() == reflect.Pointer {
		tt = tt.Elem()
	}

	for i := 0; i < tt.NumField(); i++ {
		f := tt.Field(i)

		tag, opts := parseTag(f.Tag.Get(usingTagName))
		if tag == "" {
			continue
		}

		// fast paths in Unmarshal may bypass these instructions for common types
		assigner := reflectKindToAssigner(f.Type, usingTagName, opts, d.limits)

		// route decode instructions to trie or map based on name length
		// for optimal lookup performance during schema parsing
		df := decodeInstruction{fun: assigner.fun, offset: f.Offset, kind: assigner.wire, subdec: assigner.subDecoder, subType: f.Type, tag: tag, subinstr: nil, optimizable: false}
		if len(tag) < smallKeys {
			d.trie.Add(tag, df)
		} else {
			d.lookup[tag] = df
		}
		d.numfield++
	}

	return d
}

// Unmarshal Errors
var (
	ErrInvalidDocument = errors.New("invalid glint document")
	ErrSchemaNotFound  = errors.New("schema parse error. document was supplied with no schema and there are no cached instructions for the hash")
)

// DecoderContext supports trusted schema mode with an instruction cache and caller-defined affinity ID
type DecoderContext struct {
	InstructionCache *DecodeInstructionLookup
	ID               uint
	// Warning: non-static fields here cause allocations when passed to function pointers.
	// Verify with escape analysis and benchmarks before adding fields.
}

// Unmarshal populates the provided struct with data from a glint document.
// Requirements: s must be a pointer and its type must exactly match what newDecoder received.
func (d *decoderImpl) Unmarshal(bytes []byte, s any) error {
	return d.UnmarshalWithContext(bytes, s, DecoderContext{InstructionCache: &d.cache})

}

func (d *decoderImpl) UnmarshalWithContext(bytes []byte, s any, context DecoderContext) error {

	if len(bytes) < 5 {
		return ErrInvalidDocument
	}

	if d.numfield == 0 {
		return nil
	}

	// Reader traverses the document using value semantics (not pointers) to ensure stack allocation.
	// Function pointers prevent escape analysis from proving pointer safety, so we pass/return
	// by value (similar to append) to avoid heap allocation.
	r := NewReader(bytes)

	var _ = r.ReadByte() // flags
	hash := r.Read(4)
	schema := NewReader(r.Read(uint(r.ReadVarint())))
	body := NewReader(r.Remaining())

	atomic.StoreUint32(&d.lastHash, binary.LittleEndian.Uint32(hash))

	var err error
	var instructions []decodeInstruction // the full list of instructions needed to decode the given schema, including skips

	ins, okl := context.InstructionCache.get(hash) // do we have a cached set of instructions?
	if okl {
		instructions = ins

		goto start_values
	} else {

		if schema.BytesLeft() == 0 {
			return ErrSchemaNotFound
		}

		ins := [10]decodeInstruction{} // fixed-size array for stack allocation; must remain in this scope for performance
		instructions = ins[:0]
	}

	instructions, _, err = d.parseSchema(schema, instructions)
	if err != nil {
		return err
	}
	if !okl {
		context.InstructionCache.add(hash, instructions, context.ID) // cache per session for reuse
	}

start_values:
	body = d.unmarshal(body, instructions, s)

	if len(body.Remaining()) > 0 {
		return fmt.Errorf("body bytes remaining > 0: %v", len(body.Remaining()))
	}

	return nil
}

// parseSchema transforms the received schema into an ordered instruction list using our pre-built lookups
func (d *decoderImpl) parseSchema(schema Reader, instructions []decodeInstruction) ([]decodeInstruction, Reader, error) {

start_schema:
	// Build execution order by matching schema field names to our stored decoder functions.
	// The resulting instruction array can be cached for future documents with the same schema.

	if schema.BytesLeft() == 0 {
		return instructions, schema, nil
	}

	// each schema entry has 3 core elements - specialized types may include extra data
	wireType := WireType(schema.ReadVarint())
	nameLen := schema.ReadByte()
	name := schema.Read(uint(nameLen))

	var di decodeInstruction
	var ok bool
	if len(name) < smallKeys { // fast path for small names
		di, ok = d.trie.Get(*(*string)(unsafe.Pointer(&name)))
	} else {
		di, ok = d.lookup[*(*string)(unsafe.Pointer(&name))]
	}

	// if the field name is not in the trie/lookup then we'll skip it
	if ok && di.kind != WireType(wireType) {
		return nil, schema, fmt.Errorf("schema mismatch for field %q, expected id %v got %v", name, di.kind, wireType)
	}

	if !ok {
		// unknown field in schema - create skip instruction to bypass it.
		// wireType gets signed to distinguish from actual body instructions.

		switch {
		case wireType&^WirePtrFlag == WireStruct:
			// unknown struct field - build temporary decoder to navigate past all its sub-fields
			dec := newDecoder(struct{}{})
			sl := schema.ReadVarint()

			ins, _, err := dec.parseSchema(NewReader(schema.Read(sl)), nil) // parse unwanted object's schema for skipping
			if err != nil {
				return nil, schema, err
			}

			skipfun := func(p unsafe.Pointer, r Reader) Reader {
				return dec.unmarshal(r, ins, struct{}{}) // discard data by decoding to empty struct
			}
			if wireType&WirePtrFlag > 0 {
				skipfun = deref(skipfun, wireType, di.subType)
			}
			instructions = append(instructions, decodeInstruction{fun: skipfun, tag: string(name), optimizable: false})

		case wireType&WireSliceFlag > 0:

			dec := sliceDecoder{wireType: wireType}

			var err error
			_, schema, err = dec.parseSchema(schema, nil)
			if err != nil {
				return nil, schema, err
			}

			skipfun := func(p unsafe.Pointer, r Reader) Reader {
				return dec.unmarshal(r, nil, []struct{}{}) // discard slice data by decoding to empty slice
			}
			if wireType&WirePtrFlag > 0 {
				skipfun = deref(skipfun, wireType, di.subType)
			}
			instructions = append(instructions, decodeInstruction{fun: skipfun, tag: string(name), kind: wireType, optimizable: false})
		case wireType&WireTypeMask == WireMap:

			dec := mapDecoder{}

			var err error
			_, schema, err = dec.parseSchema(schema, nil)
			if err != nil {
				return nil, schema, err
			}

			skipfun := func(p unsafe.Pointer, r Reader) Reader {
				return dec.unmarshal(r, nil, make(map[string]struct{}))
			}
			if wireType&WirePtrFlag > 0 {
				skipfun = deref(skipfun, wireType, di.subType)
			}
			instructions = append(instructions, decodeInstruction{fun: skipfun, tag: string(name), optimizable: false})

		default:
			// figure out what to skip later on
			instructions = append(instructions, decodeInstruction{kind: wireSkip | wireType, tag: string(name), optimizable: false})
		}

		goto start_schema
	}

	switch {

	case wireType&WireSliceFlag > 0 || wireType == WireMap:
		di.subdec.setWireType(wireType)
		var err error
		_, schema, err = di.subdec.parseSchema(schema, nil)
		if err != nil {
			return nil, schema, err
		}

	case wireType == WireStruct || wireType^WirePtrFlag == WireStruct:
		schemaLen := schema.ReadVarint()
		schemaBody := schema.Read(schemaLen)

		subinstr, _, err := di.subdec.parseSchema(NewReader(schemaBody), nil) // build nested instructions from sub-decoder
		if err != nil {
			return nil, schema, err
		}

		di.fun = func(p unsafe.Pointer, r Reader) Reader {
			return di.subdec.unmarshal(r, subinstr, p)
		}
		di.subinstr = subinstr

		if wireType&WirePtrFlag > 0 { // pointer fields need dereferencing wrapper
			di.fun = deref(di.fun, wireType, di.subType)
		} else {
			di.fun = nil // enable fast path optimization
		}

	}

	instructions = append(instructions, di)

	goto start_schema
}

// unmarshal executes compiled instructions to populate the struct with data from the reader.
func (d *decoderImpl) unmarshal(body Reader, instructions []decodeInstruction, s any) Reader {

	// Execute decoder functions on the payload body using pre-compiled instructions.
	// Each function knows how to decode or skip its field, allowing the body
	// to be a continuous byte stream of values.

	p := (*iface)(unsafe.Pointer(&s)).Data

	for i := 0; i < len(instructions); i++ {
		// inlinable fast paths - const cases required for jump table optimization
		switch instructions[i].kind {
		case WireBool:
			*(*bool)(unsafe.Add(p, instructions[i].offset)) = body.ReadBool()
		case WireInt:
			*(*int)(unsafe.Add(p, instructions[i].offset)) = body.ReadInt()
		case WireInt8:
			*(*int8)(unsafe.Add(p, instructions[i].offset)) = body.ReadInt8()
		case WireInt16:
			*(*int16)(unsafe.Add(p, instructions[i].offset)) = body.ReadInt16()
		case WireInt32:
			*(*int32)(unsafe.Add(p, instructions[i].offset)) = body.ReadInt32()
		case WireInt64:
			*(*int64)(unsafe.Add(p, instructions[i].offset)) = body.ReadInt64()
		case WireUint:
			*(*uint)(unsafe.Add(p, instructions[i].offset)) = body.ReadUint()
		case WireUint8:
			*(*uint8)(unsafe.Add(p, instructions[i].offset)) = body.ReadUint8()
		case WireUint16:
			*(*uint16)(unsafe.Add(p, instructions[i].offset)) = body.ReadUint16()
		case WireUint32:
			*(*uint32)(unsafe.Add(p, instructions[i].offset)) = body.ReadUint32()
		case WireUint64:
			*(*uint64)(unsafe.Add(p, instructions[i].offset)) = body.ReadUint64()
		case WireFloat32:
			*(*float32)(unsafe.Add(p, instructions[i].offset)) = body.ReadFloat32()
		case WireFloat64:
			*(*float64)(unsafe.Add(p, instructions[i].offset)) = body.ReadFloat64()
		case WireString:
			l := body.ReadVarint()
			if l > body.BytesLeft() {
				panic(fmt.Sprintf("string length %d exceeds remaining bytes %d", l, body.BytesLeft()))
			}

			b := body.Read(l)
			*(*string)(unsafe.Add(p, instructions[i].offset)) = *(*string)(unsafe.Pointer(&b))
		case WireTime:
			*(*time.Time)(unsafe.Add(p, instructions[i].offset)) = body.ReadTime()
		case WireStruct:
			body = instructions[i].subdec.unmarshal(body, instructions[i].subinstr, unsafe.Add(p, instructions[i].offset))
		default:
			goto dyn
		}
		continue
	dyn:

		// fallback path: check for fast path availability, otherwise use function pointer or skip
		switch {
		case instructions[i].fun != nil:
			body = instructions[i].fun(unsafe.Add(p, instructions[i].offset), body)

		case instructions[i].kind&wireSkip > 0:

			if instructions[i].kind&WirePtrFlag > 0 {
				if body.ReadByte() == 0 {
					continue
				}
			}

			switch instructions[i].kind & WireTypeMask {

			case WireInt, WireInt16, WireInt32, WireInt64,
				WireUint, WireUint16, WireUint32, WireUint64,
				WireFloat32, WireFloat64:

				body.SkipVarint()

			case WireString, WireBytes, WireTime:
				body.Skip(body.ReadVarint())

			case WireBool, WireInt8, WireUint8:
				body.Skip(1)

			default:
				panic(fmt.Sprintf("unknown skip type %v", instructions[i].kind&WireTypeMask))
			}
		default:
			panic(fmt.Sprintf("unknown instruction %v", instructions[i].kind))
		}

	}

	return body
}
