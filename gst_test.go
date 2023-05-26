package gstpl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoopCtx(t *testing.T) {
	require := require.New(t)
	require.Equal(0, loopCtxRefCnt())

	{
		pl, err := NewPipeline("fakesrc")
		require.NoError(err)
		require.Equal(1, loopCtxRefCnt())

		pl.Close()
		require.Equal(0, loopCtxRefCnt())

		pl.Close()
		require.Equal(0, loopCtxRefCnt())
	}

	{
		pl, err := NewPipeline("fakesrc")
		require.NoError(err)
		require.Equal(1, loopCtxRefCnt())

		err = pl.Start()
		require.NoError(err)
		require.Equal(1, loopCtxRefCnt())

		err = pl.Start()
		require.NoError(err)
		require.Equal(1, loopCtxRefCnt())

		pl.Close()
		require.Equal(0, loopCtxRefCnt())

		pl.Close()
		require.Equal(0, loopCtxRefCnt())
	}

	{
		pl1, err := NewPipeline("fakesrc")
		require.NoError(err)
		require.Equal(1, loopCtxRefCnt())

		pl2, err := NewPipeline("fakesrc")
		require.NoError(err)
		require.Equal(2, loopCtxRefCnt())

		pl1.Close()
		require.Equal(1, loopCtxRefCnt())

		pl2.Close()
		require.Equal(0, loopCtxRefCnt())
	}
}
