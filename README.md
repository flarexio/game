## Sample Video

```bash
ffmpeg -re -stream_loop -1 -i input.mp4 \
  -map 0:v -c:v copy -an -f h264 unix:///tmp/stream/video.sock \
  -map 0:a -vn -c:a libopus -ac 2 -page_duration 20000 -f ogg unix:///tmp/stream/audio.sock
```