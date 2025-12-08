#include <stdlib.h>
#include <stdint.h>
#include "emberplus.h"

extern void go_onNode(GlowNode *pNode, GlowFieldFlags fields, berint *pPath, int pathLength, voidptr state);
extern void go_onParameter(GlowParameter* param, GlowFieldFlags fields, berint *pPath, int pathLength, voidptr state);
extern void go_onCommand(GlowCommand* param, GlowFieldFlags fields, berint *pPath, int pathLength, voidptr state);
extern void go_onFunction(GlowFunction *pFunction, berint *pPath, int pathLength, voidptr state);
extern void go_onMatrix(GlowMatrix *pMatrix, berint *pPath, int pathLength, voidptr state);
extern void go_onStreamEntry(
GlowStreamEntry* entry, GlowFieldFlags fields, berint *pPath, int pathLength, voidptr state
);

extern void go_onLastPackageReceived(int length, void* state);
extern void go_onThrowError(int error, const char* pMessage);
extern void go_onFailAssertion(const char* pFileName, int lineNumber);

void c_onNode(const GlowNode *pNode, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state) {
	go_onNode((GlowNode *)pNode, fields, (berint *)pPath, pathLength, state);
}

void c_onParameter(const GlowParameter *pParameter, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state) {
	go_onParameter((GlowParameter *)pParameter, fields, (berint *)pPath, pathLength, state);
}

void c_onCommand(const GlowCommand* cmd, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state) {
	go_onCommand((GlowCommand *)cmd, fields, (berint *)pPath, pathLength, state);
}

void c_onStreamEntry(
const GlowStreamEntry *entry, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state
) {
	go_onStreamEntry((GlowStreamEntry *)entry, fields, (berint *)pPath, pathLength, state);
}

void c_onFunction(const GlowFunction *pFunction, const berint *pPath, int pathLength, voidptr state) {
	go_onFunction((GlowFunction *)pFunction, (berint *)pPath, pathLength, state);
}
void c_onMatrix(const GlowMatrix *pMatrix, const berint *pPath, int pathLength, voidptr state) {
	go_onMatrix((GlowMatrix *)pMatrix, (berint *)pPath, pathLength, state);
}

void c_onLastPackageReceived(const byte *pPackage, int length, voidptr state)
{
    go_onLastPackageReceived(length, state);
}

void c_onThrowError(int error, pcstr pMessage)
{
go_onThrowError(error, pMessage);
}

void c_onFailAssertion(pcstr pFileName, int lineNumber)
{
go_onFailAssertion(pFileName, lineNumber);
}


void *allocMemoryImpl(size_t size)
{
   return malloc(size);
}

void freeMemoryImpl(void *pMemory)
{
   free(pMemory);
}
