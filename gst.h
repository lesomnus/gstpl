#pragma once

#include <stdbool.h>

#include <glib.h>
#include <gst/gst.h>

typedef struct Context {
	void*       handler;
	GstElement* pipeline;
} Context;

extern void goHandleEndOfStream(void* handler);
extern void goHandleSample(void* handler, void* buffer, int bufferLen, int duration);
extern void goHandleError(void* handler, GError* error);

bool     gstpl_init(GError** err);
int      gstpl_ref_cnt();
Context* gstpl_ctx_new(char* expr, GError** error);
void     gstpl_ctx_free(Context* ctx);
void     gstpl_ctx_start(Context* ctx);
