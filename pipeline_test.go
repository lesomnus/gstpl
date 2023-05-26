package gstpl_test

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/lesomnus/gstpl"
	"github.com/stretchr/testify/require"
)

func TestNewPipeline(t *testing.T) {
	require := require.New(t)

	_, err := gstpl.NewPipeline("invalid-pipeline")
	require.Error(err)

	pl, err := gstpl.NewPipeline("videotestsrc")
	require.NoError(err)

	err = pl.Close()
	require.NoError(err)
}

func TestPipelinePlay(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("videotestsrc")
	require.NoError(err)

	err = pl.Start()
	require.NoError(err)

	sample, err := pl.Recv()
	require.NoError(err)
	require.NotEmpty(sample.Data)

	sample, err = pl.Recv()
	require.NoError(err)
	require.NotEmpty(sample.Data)

	sample, err = pl.Recv()
	require.NoError(err)
	require.NotEmpty(sample.Data)

	err = pl.Close()
	require.NoError(err)
}

func TestPipelineRecv(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("fakesrc num-buffers=5 sizetype=fixed sizemax=42")
	require.NoError(err)

	err = pl.Start()
	require.NoError(err)

	for {
		sample, err := pl.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(err)
		require.Len(sample.Data, 42)
	}

	err = pl.Close()
	require.NoError(err)
}

func TestPipelineCloseWhilePlaying(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("videotestsrc")
	require.NoError(err)

	t0 := time.Now()
	go func() {
		time.Sleep(100 * time.Millisecond)
		pl.Close()
	}()

	pl.Start()
	for {
		sample, err := pl.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(err)
		require.NotEmpty(sample.Data)
	}

	elapsed := time.Since(t0)
	require.Greater(elapsed.Milliseconds(), int64(99))
	require.Less(elapsed.Milliseconds(), int64(150))
}

func TestPipelineError(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("filesrc location=not-exists.mp4")
	require.NoError(err)
	defer pl.Close()

	t0 := time.Now()
	go func() {
		time.Sleep(100 * time.Millisecond)
		pl.Start()
	}()

	_, err = pl.Recv()
	require.Error(err)
	require.GreaterOrEqual(time.Since(t0).Milliseconds(), int64(100))

	_, err = pl.Recv()
	require.Error(err)
}

func TestPipelineStartTwice(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("videotestsrc")
	require.NoError(err)
	defer pl.Close()

	err = pl.Start()
	require.NoError(err)

	err = pl.Start()
	require.NoError(err)
}

func TestMultiplePipelines(t *testing.T) {
	play := func(require *require.Assertions, pl gstpl.Pipeline) {
		err := pl.Start()
		require.NoError(err)

		for i := 0; i < 10; i++ {
			sample, err := pl.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			require.NoError(err)
			require.NotEmpty(sample.Data)
		}

		err = pl.Close()
		require.NoError(err)
	}

	t.Run("sequential", func(t *testing.T) {
		require := require.New(t)

		{
			pl, err := gstpl.NewPipeline("videotestsrc")
			require.NoError(err)

			play(require, pl)
		}

		{
			pl, err := gstpl.NewPipeline("videotestsrc")
			require.NoError(err)

			play(require, pl)
		}
	})

	t.Run("parallel", func(t *testing.T) {
		require := require.New(t)

		pl1, err := gstpl.NewPipeline("videotestsrc")
		require.NoError(err)

		pl2, err := gstpl.NewPipeline("videotestsrc")
		require.NoError(err)

		var (
			t1a, t1b,
			t2a, t2b time.Time
			wg sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			t1a = time.Now()
			play(require, pl1)
			t1b = time.Now()
		}()
		go func() {
			defer wg.Done()
			t2a = time.Now()
			play(require, pl2)
			t2b = time.Now()
		}()
		wg.Wait()

		require.True(t1b.After(t1a))
		require.True(t1b.After(t2a))
		require.True(t2b.After(t1a))
		require.True(t2b.After(t2a))
	})

	t.Run("parallel with one paused", func(t *testing.T) {
		require := require.New(t)

		// 33.3ms per frame.
		pl1, err := gstpl.NewPipeline("videotestsrc ! video/x-raw,framerate=30/1")
		require.NoError(err)

		pl2, err := gstpl.NewPipeline("videotestsrc num-buffers=2 ! video/x-raw,framerate=30/1")
		require.NoError(err)

		var wg sync.WaitGroup
		defer wg.Wait()

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pl1.Start()
			require.NoError(err)

			// One blocked (no-recv) does not block the other pipelines.
			time.Sleep(100 * time.Millisecond)
		}()

		t0 := time.Now()
		play(require, pl2)
		require.Less(time.Since(t0).Milliseconds(), int64(100))
	})
}

func TestStartClosedPipeline(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("videotestsrc")
	require.NoError(err)

	pl.Close()

	err = pl.Start()
	require.ErrorIs(err, io.ErrClosedPipe)
}

func TestRecvFromClosedPipeline(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("videotestsrc")
	require.NoError(err)

	pl.Close()

	_, err = pl.Recv()
	require.ErrorIs(err, io.EOF)
}

func TestPipelineEndOfStream(t *testing.T) {
	require := require.New(t)

	pl, err := gstpl.NewPipeline("videotestsrc num-buffers=5")
	require.NoError(err)
	defer pl.Close()

	err = pl.Start()
	require.NoError(err)

	i := 0
	for ; i < 10; i++ {
		var sample gstpl.Sample
		sample, err = pl.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		require.NoError(err)
		require.NotEmpty(sample.Data)
	}
	require.ErrorIs(err, io.EOF)
	require.Equal(5, i)
}
