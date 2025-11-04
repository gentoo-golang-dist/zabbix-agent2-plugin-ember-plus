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

static volatile int allocCount = 0;

void *allocMemoryImpl(size_t size)
{
   void *pMemory = malloc(size);

   //if(sizeof(void *) == 8)
   //   printf("allocate %lu bytes: %llX\n", size, (unsigned long long)pMemory);
   //else
   //   printf("allocate %lu bytes: %lX\n", size, (unsigned long)pMemory);

   allocCount++;
   return pMemory;
}

void freeMemoryImpl(void *pMemory)
{
   //if(sizeof(void *) == 8)
   //   printf("free: %llX\n", (unsigned long long)pMemory);
   //else
   //   printf("free: %lX\n", (unsigned long)pMemory);

   allocCount--;
   free(pMemory);
}
