#nyaa, a terminal interface for YouTube

Do you want to watch YouTube, but you don't want to load up a huge browser to view their enormous JS site? Well, nyaa is a potential solution.

Utilizing RSS, mpv, and yt-dlp, nyaa can perform YouTube searches, manage your subscriptions, and show you a chronological feed of the new videos from the channels you're following. Your subs are saved in ~/.config/nyaa/ and that's the only file that's written to your disk other than the executable.

It's small, it's simple, and it's written in Go for maximum compatibility. It should work fine on Linux, Windows, Mac, as long as you have `mpv` and `yt-dlp` installed.

To compile you will need Go as well.

```
git clone https://github.com/RubyMaelstrom/nyaa
cd nyaa
go build
```

Then just run the nyaa binary in the directory. That's it!

Enjoy! 💖
