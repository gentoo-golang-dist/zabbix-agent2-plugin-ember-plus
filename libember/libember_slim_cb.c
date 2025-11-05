#include <stdlib.h>
#include <stdint.h>
#include "emberplus.h"

GlowReader g_reader;

extern void go_onNode(void* state, GlowNode* node);
extern void go_onParameter(void* state, GlowParameter* param);
extern void go_onLastPackageReceived(int length, void* state);

void c_onNode(const GlowNode *pNode, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state) {
	go_onNode(state, (GlowNode *)pNode);
}

void c_onParameter(const GlowParameter *pParameter, GlowFieldFlags fields, const berint *pPath, int pathLength, voidptr state) {
	go_onParameter(state, (GlowParameter *)pParameter);
}

void c_onCommand(const GlowCommand *pCommand, const berint *pPath, int pathLength, voidptr state) {
}

void c_onStreamEntry(const GlowStreamEntry *pStreamEntry, voidptr state) {
}


void c_onLastPackageReceived(const byte *pPackage, int length, voidptr state)
{
    go_onLastPackageReceived(length, state);
}

void onThrowError(int error, pcstr pMessage)
{
   printf("ber error %d: '%s'\n", error, pMessage);
}

void onFailAssertion(pcstr pFileName, int lineNumber)
{
   printf("Debug assertion failed @ '%s' line %d", pFileName, lineNumber);
}


void *allocMemoryImpl(size_t size)
{
   return malloc(size);
}

void freeMemoryImpl(void *pMemory)
{
   free(pMemory);
}
