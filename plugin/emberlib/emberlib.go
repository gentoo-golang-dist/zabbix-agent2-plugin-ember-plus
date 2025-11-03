/*
** Copyright (C) 2001-2025 Zabbix SIA
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

   #cgo CFLAGS: -I../../libember_slim/Source
   #cgo LDFLAGS: ./libember_slim/Source/libember_slim-static.a
   #cgo LDFLAGS: ./libember_slim/Source/go-sample/reader_callbacks.a

   #include <stdlib.h>
   #include <stdint.h>
   #include <string.h>
   #include "emberplus.h"


   // The library sometimes uses 'byte' typedefs. Make sure it is available:
   typedef unsigned char byte;

   // Global reader instance (single instance)
   extern GlowReader g_reader;

   // Extern declarations of Go functions we will export to C.
   // extern void go_onNode(void* state, GlowNode* node);
   // extern void go_onParameter(void* state, GlowParameter* param);
   // extern void go_onCommand(void* state, GlowCommand* cmd);
   // extern void go_onStreamEntry(void* state, GlowStreamEntry* entry);
   // extern void go_onLastPackageReceived(const byte *pPackage, int length, voidptr state);

   // C wrapper functions that match the expected onNode_t etc. signatures.
   // They simply call the exported Go functions.

   extern void c_onLastPackageReceived(const byte *pPackage, int length, voidptr state);

   extern void c_onNode(const GlowNode *pNode, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state);
   // void c_onNode(const GlowNode *pNode, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state) {
   // 	go_onNode(state, (GlowNode *)pNode);
   // }
   extern void c_onParameter(const GlowParameter *pParameter, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state);
   // void c_onParameter(const GlowParameter *pParameter, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state) {
   // 	go_onParameter(state, (GlowParameter *)pParameter);
   // }
   // static void c_onCommand(void* state, GlowCommand* cmd) {
   //     go_onCommand(state, cmd);
   // }
   // static void c_onStreamEntry(void* state, GlowStreamEntry* entry) {
   //     go_onStreamEntry(state, entry);
   // }

   // Return pointer to global reader
   static GlowReader* get_reader() {
   	return &g_reader;
   }

   extern void onThrowError(int error, pcstr pMessage);
   extern void onFailAssertion(pcstr pFileName, int lineNumber);
   extern void *allocMemoryImpl(size_t size);
   extern void freeMemoryImpl(void *pMemory);

*/
import "C"

import (
	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/plugin/ember-plus/ember/asn1"
	"golang.zabbix.com/sdk/errs"
	"net"
	"runtime/cgo"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

type EmberLib struct {
	rxBuf  unsafe.Pointer
	handle cgo.Handle
	reader *C.GlowReader
	state  *ReaderState
}

// ParsedElement stores a JSON-serializable representation of a Glow element.
type ParsedElement struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier,omitempty"`
	Path       []int  `json:"path,omitempty"`
	// extend with Value, Children, etc. as needed
}

type ReaderState struct {
	parsed   ember.ElementCollection
	parsedMu sync.Mutex
	stop     bool
}

func (p *EmberLib) EmberStart() error {
	const rxBufferSize = 8192
	p.rxBuf = C.malloc(C.size_t(rxBufferSize))
	if p.rxBuf == nil {
		return errs.New("C.malloc failed")
	}

	// Get pointer to global reader
	p.reader = C.get_reader()

	p.state = &ReaderState{
		parsed: make(ember.ElementCollection),
	}

	p.handle = cgo.NewHandle(p.state)

	C.ember_init(
		(C.throwError_t)(unsafe.Pointer(C.onThrowError)),
		(C.failAssertion_t)(unsafe.Pointer(C.onFailAssertion)),
		(C.allocMemory_t)(unsafe.Pointer(C.allocMemoryImpl)),
		(C.freeMemory_t)(unsafe.Pointer(C.freeMemoryImpl)))

	// Initialize reader with the C wrapper callbacks and our state pointer.
	// We'll pass a nil state pointer (could be used to pass Go context if marshalled properly).
	C.glowReader_init(
		p.reader,
		(C.onNode_t)(unsafe.Pointer(C.c_onNode)),           // onNode
		(C.onParameter_t)(unsafe.Pointer(C.c_onParameter)), // onParameter
		nil,                                                // (C.onCommand_t)(unsafe.Pointer(C.c_onCommand)),         // onCommand
		nil,                                                // (C.onStreamEntry_t)(unsafe.Pointer(C.c_onStreamEntry)), // onStreamEntry
		(C.voidptr)(unsafe.Pointer(p.handle)),
		(*C.byte)(unsafe.Pointer(p.rxBuf)),
		C.uint(rxBufferSize))

	p.reader.onLastPackageReceived = (C.onPackageReceived_t)(unsafe.Pointer(C.c_onLastPackageReceived))

	return nil
}

func (p *EmberLib) EmberStop() {
	C.free(p.rxBuf)
	p.handle.Delete()
}

func (p *EmberLib) EmberRead(buf []byte, n int) bool {
	C.glowReader_readBytes(p.reader, (*C.byte)(unsafe.Pointer(&buf[0])), C.int(n))
	if p.state.stop {
		return true
	}

	return false
}

func (p *EmberLib) GetFromState() ember.ElementCollection {
	p.state.parsedMu.Lock()
	out := p.state.parsed
	defer p.state.parsedMu.Unlock()

	return out
}

// sendGetDirectory encodes a GetDirectory request and writes to conn.
// This function expects a C glowWriter API. If your version uses different names,
// adapt calls below or implement your own encoder.
func SendGetDirectory(conn net.Conn, path string, request string) error {
	// Simple approach: create C tx buffer and use glowWriter_* functions if present.
	const txSize = 2048
	txBuf := C.malloc(C.size_t(txSize))
	if txBuf == nil {
		return errs.New("C.malloc failed, for txBuf")
	}
	defer C.free(txBuf)

	// initialize writer on txBuf
	var writer C.GlowOutput
	C.glowOutput_init(&writer, (*C.byte)(txBuf), C.uint(txSize), 0)

	// Split path
	parts := splitPath(path)
	pathLen := (C.int)(len(parts))
	pathBuffSize := C.size_t(pathLen * C.int(unsafe.Sizeof(C.int(0))))
	pathBuff := C.malloc(pathBuffSize)
	if pathBuff == nil {
		return errs.New("C.malloc failed, pathBuff")
	}
	defer C.free(pathBuff)

	cParts := (*[1 << 30]C.int)(pathBuff)[:pathLen:pathLen]
	for i, p := range parts {
		num, err := strconv.Atoi(p)
		if err != nil {
			return errs.Wrap(err, "failed to parse parts")
		}
		cParts[i] = C.int(num)
	}

	C.glowOutput_beginPackage(&writer, C.true)
	var command C.GlowCommand
	// bzero_item(command)
	command.number = C.GlowCommandType_GetDirectory

	//dirFieldMaskPtr := (*C.GlowFieldFlags)(unsafe.Pointer(&command.options[0]))
	//
	//// Set it
	//*dirFieldMaskPtr = C.GlowFieldFlag_All

	switch request {
	case asn1.NodeType:
		// Build GetDirectory command
		C.glow_writeQualifiedCommand(
			&writer,
			&command,
			(*C.berint)(pathBuff), //cArray,
			pathLen,
			C.GlowElementType_Node)
	case asn1.ParameterType:
		// Build GetDirectory command
		C.glow_writeQualifiedCommand(
			&writer,
			&command,
			(*C.berint)(pathBuff), //cArray,
			pathLen,
			C.GlowElementType_Parameter)
	}

	length := C.glowOutput_finishPackage(&writer)
	if length == 0 {
		return errs.New("glowWriter_getLength returned 0")
	}

	// Create Go slice referencing txBuf
	txSlice := ((*[1 << 20]byte)(unsafe.Pointer(txBuf)))[:int(length):int(length)]
	_, err := conn.Write(txSlice)
	if err != nil {
		return errs.Wrap(err, "failed to write GetDirectory request")
	}

	return nil
}

//export go_onLastPackageReceived
func go_onLastPackageReceived(length int, state unsafe.Pointer) {
	h := cgo.Handle(state)
	rstate, ok := h.Value().(*ReaderState)
	if !ok {
		return
	}
	rstate.stop = true
}

//export go_onNode
func go_onNode(node *C.GlowNode, _ *C.GlowFieldFlags, pPath *C.berint, pathLength int, state unsafe.Pointer) {
	h := cgo.Handle(state)
	rstate, ok := h.Value().(*ReaderState)
	if !ok {
		return
	}

	// Access node->identifier (assumes char* identifier)
	id := ""
	if node != nil && node.pIdentifier != nil {
		id = C.GoString(node.pIdentifier)
	}

	pathC := unsafe.Slice(pPath, pathLength)
	pathGo := make([]string, pathLength)
	for i, v := range pathC {
		pathGo[i] = strconv.Itoa(int(v))
	}

	strPath := strings.Join(pathGo, ".")

	k := ember.ElementKey{
		ID:   id,
		Path: strPath,
	}

	rstate.parsedMu.Lock()
	rstate.parsed[k] = &ember.Element{
		Path:        strPath,
		ElementType: asn1.NodeType,
		Identifier:  id,
	}
	rstate.parsedMu.Unlock()
}

//export go_onParameter
func go_onParameter(param *C.GlowParameter, _ *C.GlowFieldFlags, pPath *C.berint, pathLength int, state unsafe.Pointer) {
	h := cgo.Handle(state)
	rstate, ok := h.Value().(*ReaderState)
	if !ok {
		return
	}

	id := ""
	if param != nil && param.pIdentifier != nil {
		id = C.GoString(param.pIdentifier)
	}

	pathC := unsafe.Slice(pPath, pathLength)
	pathGo := make([]string, pathLength)
	for i, v := range pathC {
		pathGo[i] = strconv.Itoa(int(v))
	}

	strPath := strings.Join(pathGo, ".")

	k := ember.ElementKey{
		ID:   id,
		Path: strPath,
	}

	rstate.parsedMu.Lock()
	rstate.parsed[k] = &ember.Element{
		Path:        strPath,
		ElementType: asn1.ParameterType,
		Identifier:  id,
	}

	rstate.parsedMu.Unlock()
}

//export go_onCommand
func go_onCommand(state unsafe.Pointer, cmd *C.GlowCommand) {
	// Minimal handling; extend as needed
	// parsedMu.Lock()
	// parsed = append(parsed, ParsedElement{Type: "Command"})
	// parsedMu.Unlock()
}

//export go_onStreamEntry
func go_onStreamEntry(state unsafe.Pointer, entry *C.GlowStreamEntry) {
	// parsedMu.Lock()
	// parsed = append(parsed, ParsedElement{Type: "StreamEntry"})
	// parsedMu.Unlock()
}

// splitPath splits an Ember path "root/audio/volume" into parts (no empty parts)
func splitPath(path string) []string {
	var res []string
	start := 0
	for i := 0; i < len(path); i++ {
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
