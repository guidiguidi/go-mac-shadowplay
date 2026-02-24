package recorder

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/guidiguidi/go-mac-shadowplay/internal/buffer"
	"github.com/tfsoares/screencapturekit-go"
)

type VideoFrameHeader struct {
	Width       uint32
	Height      uint32
	BytesPerRow uint32
	Timestamp   float64
	DataSize    uint32
}

type ShadowRecorder struct {
	recorder *screencapturekit.ScreenCaptureKit
	buffer   *buffer.RingBuffer
	active   bool
	mu       sync.Mutex
	pipePath string
}

func NewShadowRecorder(bufferSize int) (*ShadowRecorder, error) {
	sck, err := screencapturekit.NewScreenCaptureKit()
	if err != nil {
		return nil, err
	}

	pipePath := filepath.Join(os.TempDir(), "shadowplay_video.fifo")
	os.Remove(pipePath)
	if err := syscall.Mkfifo(pipePath, 0666); err != nil {
		return nil, fmt.Errorf("pipe error: %w", err)
	}

	return &ShadowRecorder{
		recorder: sck,
		buffer:   buffer.New(bufferSize),
		pipePath: pipePath,
	}, nil
}

func (r *ShadowRecorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active {
		return nil
	}

	screens, err := screencapturekit.GetScreens()
	if err != nil {
		return err
	}

	options := screencapturekit.StreamingOptions{
		FPS:                    60,
		ScreenID:               screens[0].ID,
		StreamingEnabled:       true,
		StreamingProtocol:      "pipe",
		StreamingVideoPipePath: &r.pipePath,
		StreamVideo:            true,
	}

	go r.readLoop()

	if err := r.recorder.StartStreaming(options); err != nil {
		return err
	}

	r.active = true
	return nil
}

func (r *ShadowRecorder) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.active {
		return nil
	}

	r.recorder.StopStreaming()
	r.active = false
	return nil
}

func (r *ShadowRecorder) readLoop() {
	pipe, err := os.OpenFile(r.pipePath, os.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer pipe.Close()

	for {
		var header VideoFrameHeader
		if err := binary.Read(pipe, binary.LittleEndian, &header); err != nil {
			break
		}

		data := make([]byte, header.DataSize)
		if _, err := io.ReadFull(pipe, data); err != nil {
			break
		}

		r.buffer.Put(buffer.Frame{
			Data:      data,
			Timestamp: int64(header.Timestamp * 1e9),
		})
	}
}

func (r *ShadowRecorder) Save(outputPath string) error {
	r.mu.Lock()
	count := r.buffer.Len()
	if count == 0 {
		r.mu.Unlock()
		return fmt.Errorf("empty buffer")
	}

	frames := make([]buffer.Frame, 0, count)
	for i := 0; i < count; i++ {
		if f, ok := r.buffer.Get(); ok {
			frames = append(frames, f)
		}
	}
	r.mu.Unlock()

	// Hardware-accelerated encoding (h264_videotoolbox)
	cmd := exec.Command("ffmpeg",
		"-f", "rawvideo",
		"-pixel_format", "bgra",
		"-video_size", "3024x1890", // FIXME: detect resolution
		"-framerate", "60",
		"-i", "-",
		"-c:v", "h264_videotoolbox", // Use Apple hardware accel
		"-b:v", "10M",
		"-pix_fmt", "yuv420p",
		"-y", outputPath,
	)

	stdin, _ := cmd.StdinPipe()
	cmd.Start()

	go func() {
		defer stdin.Close()
		for _, f := range frames {
			stdin.Write(f.Data.([]byte))
		}
	}()

	return cmd.Wait()
}
