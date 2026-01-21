/*
** Copyright (C) 2001-2026 Zabbix SIA
**
** This program is free software: you can redistribute it and/or modify it under the terms of
** the GNU Affero General Public License as published by the Free Software Foundation, version 3.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
** without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
** See the GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License along with this program.
** If not, see <https://www.gnu.org/licenses/>.
**/

package emberlib

/*
   #cgo LDFLAGS: -lember_slim-static
   #cgo LDFLAGS: -lember_slim_cb

   #include <stdlib.h>
   #include <stdint.h>
   #include <string.h>
   #include "emberplus.h"
   #include "emberinternal.h"

   #ifdef BOOL_DEFINED
   static inline _Bool EMBER_TRUE(void)  { return (_Bool)1; }
   static inline _Bool EMBER_FALSE(void) { return (_Bool)0; }
   #else
   static inline bool EMBER_TRUE(void)  { return (bool)1; }
   static inline bool EMBER_FALSE(void) { return (bool)0; }
   #endif

   // The library sometimes uses 'byte' typedefs. Make sure it is available:
   typedef unsigned char byte;

   // Extern declarations of Go functions we will export to C.
   extern void c_onLastPackageReceived(const byte *pPackage, int length, voidptr state);

   extern void c_onNode(const GlowNode *pNode, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state);
   extern void c_onParameter(const GlowParameter *pParameter, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state);
   extern void c_onCommand(const GlowCommand *cmd, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state);
   extern void c_onStreamEntry(const GlowStreamEntry *entry, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state);
   extern void c_onFunction(const GlowFunction *pFunction, const berint *pPath, int pathLength, voidptr state);
   extern void c_onMatrix(const GlowMatrix *pMatrix, const berint *pPath, int pathLength, voidptr state);

   extern void c_onThrowError(int error, pcstr pMessage);
   extern void c_onFailAssertion(pcstr pFileName, int lineNumber);
   extern void *allocMemoryImpl(size_t size);
   extern void freeMemoryImpl(void *pMemory);

*/
import "C"

import (
	"net"
	"runtime/cgo"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
)

const (
	getDirCmd = 32
	unSubCmd  = 31
)

//nolint:gochecknoglobals // using global logger, to pass it to emberlib c function wrappers.
var emberLogger log.Logger

// Handler is the main stucter used to wrap around C data structures and readers.
// Should be created with NewHandler().
type Handler struct {
	rxBuf   unsafe.Pointer
	cHandle cgo.Handle
	reader  *C.GlowReader
	state   *ReaderState
}

// ReaderState holds information about a read state and it's mutexes.
type ReaderState struct {
	parsed   ember.ElementCollection
	parsedMu sync.Mutex
	stop     bool
}

type emberCMD int

// Init initializes global ember library with required functions and sets the global logger.
func Init(l log.Logger) {
	C.ember_init(
		C.throwError_t(C.c_onThrowError),
		C.failAssertion_t(C.c_onFailAssertion),
		C.allocMemory_t(C.allocMemoryImpl),
		C.freeMemory_t(C.freeMemoryImpl),
	)

	emberLogger = l
}

// NewHandler returns a handler with all ember lib readers and functions initialized.
func NewHandler() (*Handler, error) {
	var h Handler

	const rxBufferSize = 8192
	h.rxBuf = C.malloc(C.size_t(rxBufferSize))
	if h.rxBuf == nil {
		return nil, errs.New("C.malloc failed")
	}

	// Get pointer to global reader
	h.reader = (*C.GlowReader)(C.malloc(C.size_t(unsafe.Sizeof(C.GlowReader{}))))

	h.state = &ReaderState{
		parsed: make(ember.ElementCollection),
	}

	h.cHandle = cgo.NewHandle(h.state)

	// Initialize reader with the C wrapper callbacks and our state pointer.
	// We'll pass a nil state pointer (could be used to pass Go context if marshalled properly).
	C.glowReader_init(
		h.reader,
		C.onNode_t(C.c_onNode),
		C.onParameter_t(C.c_onParameter),
		C.onCommand_t(C.c_onCommand),
		C.onStreamEntry_t(C.c_onStreamEntry), //nolint:govet // false positive
		C.voidptr(h.cHandle),
		(*C.byte)(h.rxBuf),
		C.uint(rxBufferSize),
	)

	h.reader.onLastPackageReceived = C.onPackageReceived_t(C.c_onLastPackageReceived)
	h.reader.base.onFunction = C.onFunction_t(C.c_onFunction)
	h.reader.base.onMatrix = C.onMatrix_t(C.c_onMatrix)

	return &h, nil
}

// CleanUp cleans up c data and removes c handler.
func (h *Handler) CleanUp() {
	C.free(h.rxBuf)
	C.free(unsafe.Pointer(h.reader))
	h.cHandle.Delete()
}

// EmberRead reads data from ember glow reader, returns true is reading state is done.
// If true is returned it means that data is read into state and it can be extracted.
func (h *Handler) EmberRead(buf []byte, n int) bool {
	C.glowReader_readBytes(h.reader, (*C.byte)(unsafe.Pointer(&buf[0])), C.int(n))

	return h.state.stop
}

// TakeFromState returns element collection stored in handler state.
// Resets state stop to false, for next reads.
func (h *Handler) TakeFromState() ember.ElementCollection {
	h.state.parsedMu.Lock()
	out := h.state.parsed
	h.state.parsed = make(ember.ElementCollection)
	h.state.stop = false
	defer h.state.parsedMu.Unlock()

	return out
}

// SendGetDirectory encodes a GetDirectory request and writes to conn.
func SendGetDirectory(conn net.Conn, path, request string) error {
	return sendCommand(conn, path, request, getDirCmd)
}

// SendUnsubscribe sends an command with an unsubscribe request.
func SendUnsubscribe(conn net.Conn, path, request string) error {
	return sendCommand(conn, path, request, unSubCmd)
}

// sendCommand encodes a request and writes to conn.
//
//nolint:gocyclo,cyclop // more readable and easy to understand as a single function because of C go.
func sendCommand(conn net.Conn, path, request string, cmd emberCMD) error {
	// Simple approach: create C tx buffer and use glowWriter_* functions if present.
	const txSize = 2048
	txBuf := C.malloc(C.size_t(txSize))
	if txBuf == nil {
		return errs.New("C.malloc failed, for txBuf")
	}

	//nolint:nlreturn // false positive.
	defer C.free(txBuf)

	// initialize writer on txBuf
	var writer C.GlowOutput
	C.glowOutput_init(&writer, (*C.byte)(txBuf), C.uint(txSize), 0) //nolint:gocritic // false positive.

	// Split path
	parts := splitPath(path)
	pathLen := C.int(len(parts))
	pathBuffSize := C.size_t(pathLen * C.int(unsafe.Sizeof(C.int(0))))
	pathBuff := C.malloc(pathBuffSize)
	if pathBuff == nil {
		return errs.New("C.malloc failed, pathBuff")
	}

	//nolint:nlreturn // false positive.
	defer C.free(pathBuff)

	cParts := (*[1 << 30]C.int)(pathBuff)[:pathLen:pathLen]
	for i, p := range parts {
		num, err := strconv.Atoi(p)
		if err != nil {
			return errs.Wrap(err, "failed to parse parts")
		}
		cParts[i] = C.int(num)
	}
	C.glowOutput_beginPackage(&writer, C.EMBER_TRUE()) //nolint:gocritic // false positive.
	var command C.GlowCommand

	switch cmd {
	case getDirCmd:
		command.number = C.GlowCommandType_GetDirectory

		dirFieldMaskPtr := (*C.GlowFieldFlags)(unsafe.Pointer(&command.options[0]))
		*dirFieldMaskPtr = C.GlowFieldFlag_All
	case unSubCmd:
		command.number = C.GlowCommandType_Unsubscribe
	default:
		return errs.New("unknown command")
	}

	switch request {
	case ember.NodeType:
		// Build GetDirectory command
		C.glow_writeQualifiedCommand(
			&writer,
			&command,
			(*C.berint)(pathBuff),
			pathLen,
			C.GlowElementType_Node) //nolint:gocritic // false positive.
	case ember.ParameterType:
		C.glow_writeQualifiedCommand(
			&writer,
			&command,
			(*C.berint)(pathBuff),
			pathLen,
			C.GlowElementType_Parameter) //nolint:gocritic // false positive.
	case ember.FunctionType:
		C.glow_writeQualifiedCommand(
			&writer,
			&command,
			(*C.berint)(pathBuff),
			pathLen,
			C.GlowElementType_Function) //nolint:gocritic // false positive.
	case ember.MatrixType:
		C.glow_writeQualifiedCommand(
			&writer,
			&command,
			(*C.berint)(pathBuff),
			pathLen,
			C.GlowElementType_Matrix) //nolint:gocritic // false positive.
	}

	length := C.glowOutput_finishPackage(&writer) //nolint:gocritic,nlreturn // false positive.
	if length == 0 {
		return errs.New("glowWriter_getLength returned 0")
	}

	// Create Go slice referencing txBuf
	txSlice := ((*[1 << 20]byte)(txBuf))[:int(length):int(length)]
	_, err := conn.Write(txSlice)
	if err != nil {
		return errs.Wrap(err, "failed to write GetDirectory request")
	}

	return nil
}

//export go_onNode
func go_onNode(node *C.GlowNode, _ *C.GlowFieldFlags, pPath *C.berint, pathLength int, state unsafe.Pointer) {
	el := &ember.Element{
		ElementType: ember.NodeType,
	}

	// Access node->identifier (assumes char* identifier)
	if node != nil {
		if node.pIdentifier != nil {
			el.Identifier = C.GoString(node.pIdentifier)
		}

		if node.pDescription != nil {
			el.Description = C.GoString(node.pDescription)
		}

		if node.pSchemaIdentifiers != nil {
			el.SchemaIdentifiers = C.GoString(node.pSchemaIdentifiers)
		}

		if node.isRoot == C.EMBER_TRUE() {
			el.IsRoot = true
		}

		if node.isOnline == C.EMBER_TRUE() {
			el.IsOnline = true
		}
	}

	setPath(el, pPath, pathLength)
	setIntoState(el, state)
}

//export go_onParameter
//nolint:nestif // complexity here is fine as we are simply setting fields
func go_onParameter(
	param *C.GlowParameter,
	_ *C.GlowFieldFlags,
	pPath *C.berint,
	pathLength int,
	state unsafe.Pointer,
) {
	el := &ember.Element{
		ElementType: ember.ParameterType,
	}

	if param != nil {
		if param.pIdentifier != nil {
			el.Identifier = C.GoString(param.pIdentifier)
		}

		if param.pDescription != nil {
			el.Description = C.GoString(param.pDescription)
		}

		if param.pFormat != nil {
			el.Format = C.GoString(param.pFormat)
		}

		if param.pEnumeration != nil {
			el.Enumeration = C.GoString(param.pEnumeration)
		}

		if param.pSchemaIdentifiers != nil {
			el.SchemaIdentifiers = C.GoString(param.pSchemaIdentifiers)
		}

		el.Access = int(param.access)
		el.Factor = int(param.factor)

		el.Value, el.ValueType = glowValueToGo(&param.value)
		el.Default, _ = glowValueToGo(&param.defaultValue)
		el.Minimum = glowMinMaxToGo(&param.minimum)
		el.Maximum = glowMinMaxToGo(&param.maximum)

		if param.isOnline == C.EMBER_TRUE() {
			el.IsOnline = true
		}
	}

	setPath(el, pPath, pathLength)
	setIntoState(el, state)
}

//export go_onCommand
func go_onCommand(glow *C.GlowCommand, _ *C.GlowFieldFlags, pPath *C.berint, pathLength int, state unsafe.Pointer) {
	el := &ember.Element{
		ElementType: ember.CommandType,
		Number:      int(glow.number),
	}

	setPath(el, pPath, pathLength)
	setIntoState(el, state)
}

//export go_onStreamEntry
func go_onStreamEntry(
	entry *C.GlowStreamEntry,
	_ *C.GlowFieldFlags,
	pPath *C.berint,
	pathLength int,
	state unsafe.Pointer,
) {
	el := &ember.Element{
		ElementType: ember.StreamType,
	}

	if entry != nil {
		el.StreamIdentifier = int(entry.streamIdentifier)
		el.StreamValue, el.ValueType = glowValueToGo(&entry.streamValue)
	}

	setPath(el, pPath, pathLength)
	setIntoState(el, state)
}

//export go_onFunction
func go_onFunction(f *C.GlowFunction, pPath *C.berint, pathLength int, state unsafe.Pointer) {
	el := &ember.Element{
		ElementType: ember.FunctionType,
	}

	if f != nil {
		if f.pIdentifier != nil {
			el.Identifier = C.GoString(f.pIdentifier)
		}

		if f.pDescription != nil {
			el.Description = C.GoString(f.pDescription)
		}
	}

	setPath(el, pPath, pathLength)
	setIntoState(el, state)
}

//export go_onMatrix
func go_onMatrix(m *C.GlowMatrix, pPath *C.berint, pathLength int, state unsafe.Pointer) {
	el := &ember.Element{
		ElementType: ember.MatrixType,
	}

	if m != nil {
		if m.pIdentifier != nil {
			el.Identifier = C.GoString(m.pIdentifier)
		}

		if m.pDescription != nil {
			el.Description = C.GoString(m.pDescription)
		}
	}

	setPath(el, pPath, pathLength)
	setIntoState(el, state)
}

//export go_onLastPackageReceived
func go_onLastPackageReceived(_ int, state unsafe.Pointer) {
	h := cgo.Handle(state)
	rstate, ok := h.Value().(*ReaderState)
	if !ok {
		return
	}
	rstate.stop = true
}

//export go_onThrowError
func go_onThrowError(errCode int, message *C.char) {
	//nolint:unconvert // string() conversion is needed other Errf does not understand that it's the correct type.
	emberLogger.Errf("ember error code %d: %s", errCode, string(C.GoString(message)))
}

//export go_onFailAssertion
func go_onFailAssertion(fileName *C.char, line int) {
	//nolint:unconvert // string() conversion is needed other Errf does not understand that it's the correct type.
	emberLogger.Errf("ember assertion error on line %d: %s", line, string(C.GoString(fileName)))
}

// splitPath splits an Ember path into parts (no empty parts).
func splitPath(path string) []string {
	var res []string
	start := 0
	for i := range len(path) {
		if path[i] == '.' {
			if i > start {
				res = append(res, path[start:i])
			}
			start = i + 1
		}
	}

	if start < len(path) {
		res = append(res, path[start:])
	}

	return res
}

func intToBool(in int) bool {
	return in == 1
}

func setPath(el *ember.Element, pPath *C.berint, pathLength int) {
	pathC := unsafe.Slice(pPath, pathLength)
	pathGo := []string{}

	for _, v := range pathC {
		pathGo = append(pathGo, strconv.Itoa(int(v)))
	}

	el.Path = strings.Join(pathGo, ".")
}

func setIntoState(el *ember.Element, state unsafe.Pointer) {
	h := cgo.Handle(state)
	rstate, ok := h.Value().(*ReaderState)
	if !ok {
		log.Errf("failed to set state with element path: %s and id %s", el.Path, el.Identifier)

		return
	}

	k := ember.ElementKey{
		ID:   el.Identifier,
		Path: el.Path,
	}

	rstate.parsedMu.Lock()
	rstate.parsed[k] = el
	rstate.parsedMu.Unlock()
}

//nolint:cyclop // function has a single select will all options so real way to reduce.
func glowValueToGo(val *C.GlowValue) (any, int) {
	if val == nil {
		return nil, 0
	}

	switch val.flag {
	case C.GlowParameterType_Integer:
		p := (*C.berlong)(unsafe.Pointer(&val.choice))

		return int64(*p), ember.TypeInt

	case C.GlowParameterType_Real:
		p := (*C.double)(unsafe.Pointer(&val.choice))

		return float64(*p), ember.TypeReal

	case C.GlowParameterType_Boolean:
		p := (*C.bool)(unsafe.Pointer(&val.choice))

		return *p != C.EMBER_FALSE(), ember.TypeBool

	case C.GlowParameterType_String:
		p := (**C.char)(unsafe.Pointer(&val.choice))

		return C.GoString(*p), ember.TypeString
	case C.GlowParameterType_Octets:
		octets := (*C.GlowOctetsValue)(unsafe.Pointer(&val.choice))
		length := int(octets.length)

		//nolint:nlreturn // false positive.
		return C.GoBytes(unsafe.Pointer(octets.pOctets), C.int(length)), ember.TypeOctets
	case C.GlowParameterType_Trigger:
		return "Trigger", ember.TypeTrigger
	case C.GlowParameterType_Enum:
		p := (*C.berlong)(unsafe.Pointer(&val.choice))

		return int64(*p), ember.TypeEnum
	case C.GlowParameterType_None:
		return nil, 0
	default:
		return nil, 0
	}
}

func glowMinMaxToGo(val *C.GlowMinMax) any {
	if val == nil {
		return nil
	}

	switch val.flag {
	case C.GlowParameterType_Integer:
		p := (*C.berlong)(unsafe.Pointer(&val.choice))

		return int64(*p)
	case C.GlowParameterType_Real:
		p := (*C.double)(unsafe.Pointer(&val.choice))

		return float64(*p)
	case C.GlowParameterType_None:
		return nil
	default:
		return nil
	}
}
