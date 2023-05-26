#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <threads.h>

#include <glib.h>
#include <gst/gst.h>

#include "gst.h"

static struct {
	mtx_t  lock;
	thrd_t thrd;
	int    ref;

	GMainLoop* loop;
} gstpl_main_loop_ctx;

static int gstpl_main_loop_runner_(void* arg) {
	g_main_loop_run(gstpl_main_loop_ctx.loop);
	return 0;
}

static void gstpl_main_loop_lock_() {
	if(mtx_lock(&gstpl_main_loop_ctx.lock) != thrd_success) {
		fprintf(stderr, "failed to lock gstpl main loop.\n");
		exit(EXIT_FAILURE);
		return;
	}
}

static void gstpl_main_loop_unlock_() {
	mtx_unlock(&gstpl_main_loop_ctx.lock);
}

static void gstpl_main_loop_ref_() {
	gstpl_main_loop_lock_();

	if(gstpl_main_loop_ctx.ref++ == 0) {
		gstpl_main_loop_ctx.loop = g_main_loop_new(NULL, false);
		if(thrd_create(&gstpl_main_loop_ctx.thrd, gstpl_main_loop_runner_, NULL) != thrd_success) {
			fprintf(stderr, "failed to start gstpl main loop.\n");
			exit(EXIT_FAILURE);
			return;
		}
	}

	gstpl_main_loop_unlock_();
}

static void gstpl_main_loop_unref_() {
	gstpl_main_loop_lock_();

	if(--gstpl_main_loop_ctx.ref == 0) {
		g_main_loop_quit(gstpl_main_loop_ctx.loop);
		g_main_loop_unref(gstpl_main_loop_ctx.loop);
		gstpl_main_loop_ctx.loop = NULL;

		if(thrd_join(gstpl_main_loop_ctx.thrd, NULL) != thrd_success) {
			fprintf(stderr, "failed to join gstpl main loop.\n");
			exit(EXIT_FAILURE);
		}
	}

	gstpl_main_loop_unlock_();
}

bool gstpl_init(GError** err) {
	static gboolean ok = false;
	if(ok) {
		return ok;
	}

	gstpl_main_loop_ctx.ref  = 0;
	gstpl_main_loop_ctx.loop = NULL;
	if(mtx_init(&gstpl_main_loop_ctx.lock, mtx_plain) != thrd_success) {
		fprintf(stderr, "failed to initialize the lock.\n");
		exit(EXIT_FAILURE);
		return false;
	}

	return ok = gst_init_check(NULL, NULL, err);
}

int gstpl_ref_cnt() {
	gstpl_main_loop_lock_();
	int const cnt = gstpl_main_loop_ctx.ref;
	gstpl_main_loop_unlock_();
	return cnt;
}

static gboolean gstpl_watch_bus_(GstBus* bus, GstMessage* msg, gpointer data) {
	Context* ctx = (Context*)data;

	switch(GST_MESSAGE_TYPE(msg)) {
	case GST_MESSAGE_EOS: {
		goHandleEndOfStream(ctx->handler);
		break;
	}

	case GST_MESSAGE_ERROR: {
		gchar*  debug;
		GError* error;

		gst_message_parse_error(msg, &error, &debug);
		g_free(debug);

		goHandleError(ctx->handler, error);
		g_error_free(error);
		break;
	}

	default:
		break;
	}

	return true;
}

static GstFlowReturn gstpl_handle_sample_(GstElement* object, gpointer data) {
	Context* const ctx = (Context*)data;

	GstSample* sample = NULL;
	g_signal_emit_by_name(object, "pull-sample", &sample);
	if(sample) {
		GstBuffer* const buffer = gst_sample_get_buffer(sample);
		if(buffer) {
			gpointer copy      = NULL;
			gsize    copy_size = 0;
			gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
			goHandleSample(ctx->handler, copy, copy_size, GST_BUFFER_DURATION(buffer));
		}
		gst_sample_unref(sample);
	}

	return GST_FLOW_OK;
}

Context* gstpl_ctx_new(char* expr, GError** err) {
	GstElement* const pipeline = gst_parse_launch(expr, err);
	if(pipeline == NULL) {
		return NULL;
	}

	Context* ctx  = malloc(sizeof(Context));
	ctx->pipeline = pipeline;
	{
		GstBus* bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
		gst_bus_add_watch(bus, gstpl_watch_bus_, ctx);
		gst_object_unref(bus);
	}
	{
		GstElement* appsink = gst_bin_get_by_name(GST_BIN(pipeline), "appsink");
		g_object_set(appsink, "emit-signals", TRUE, NULL);
		g_signal_connect(appsink, "new-sample", G_CALLBACK(gstpl_handle_sample_), ctx);
		gst_object_unref(appsink);
	}

	gstpl_main_loop_ref_();
	return ctx;
}

void gstpl_ctx_free(Context* ctx) {
	gst_element_set_state(ctx->pipeline, GST_STATE_NULL);
	gst_object_unref(ctx->pipeline);
	ctx->handler  = NULL;
	ctx->pipeline = NULL;

	gstpl_main_loop_unref_();
	free(ctx);
}

void gstpl_ctx_start(Context* ctx) {
	gst_element_set_state(ctx->pipeline, GST_STATE_PLAYING);
}
