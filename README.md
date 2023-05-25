# gstpl

Simple GStreamer pipeline launcher for Go.

## Example

```go
import (
	"errors"
	"time"

	"github.com/lesomnus/gstpl"
)

func main() {
	// GStreamer pipeline expression;
	// for example:
	// expr := `v4l2src device=/dev/video0
	// ! videoconvert
	// ! video/x-raw,format=YUY2,width=640,height=480,framerate=30/1
	// ! x264enc
	// ! speed-preset=ultrafast tune=zerolatency bitrate=500
	// `
	expr := "videotestsrc"

	pl, err := gstpl.NewPipeline(expr)
	if err != nil {
		panic(err)
	}

	if err := pl.Start(); err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(10 * time.Second)

		// `Close` will unblock `Recv` with `io.EOF`.
		if err := pl.Close(); err != nil {
			panic(err)
		}
	}()
	
	for {
		sample, err := pl.Recv()
		if err != nil {
			if errors.is(err, io.EOF) {
				break
			}
			
			panic(err)
		}

		// Do cool things with `sample`.
	}
}
```

