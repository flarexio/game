# Edge Gaming

## Sample Video

```bash
ffmpeg -re -stream_loop -1 -i input.mp4 \
  -map 0:v -c:v copy -an -f h264 unix:///tmp/stream/video.sock \
  -map 0:a -vn -c:a libopus -ac 2 -page_duration 20000 -f ogg unix:///tmp/stream/audio.sock
```

## Edge Gaming

```bash
# ffmpeg -hide_banner -h encoder=h264_nvenc
ffmpeg -init_hw_device d3d11va -filter_complex "ddagrab=0:offset_x=1280:offset_y=720:video_size=1280x720:framerate=60" -c:v h264_nvenc -preset p1 -tune ull -f h264 tcp://localhost:3000

# ffmpeg -hide_banner -list_devices true -f dshow -i dummy
# ffmpeg -hide_banner -h encoder=libopus
ffmpeg -f dshow -i audio="立體聲混音 (Realtek High Definition Audio)" -ac 2 -c:a libopus -b:a 64k -application lowdelay -page_duration 2000 -f opus tcp://localhost:3002
```
