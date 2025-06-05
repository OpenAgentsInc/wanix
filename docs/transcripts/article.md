From https://progrium.xyz/blog/2025/spirit-of-plan9-on-the-web/

If you go back to the [first talk ever given on webhooks](https://www.slideshare.net/slideshow/web-hooks/263894), it opens on the command-line. Specifically the Unix shell, focusing on one of its defining features: pipes. The idea was that pipes brought a new level of compositionality to programs, and webhooks could bring a new level of compositionality to web apps. Perhaps you could say I was trying to bring the spirit of Unix to the web.

With this [last release of Wanix](https://github.com/tractordev/wanix/releases/tag/v0.3-preview), I'm at it again. This time with the successor to Unix, a little known operating system called [Plan 9 from Bell Labs](https://en.wikipedia.org/wiki/Plan_9_from_Bell_Labs).

Plan 9 has been on my mind for quite a while. In fact, around the time of that first talk on webhooks, the team behind Unix and Plan 9 was being re-assembled to create the Go programming language. I pretty instantly fell in love with the Go worldview, which turns out to be an outgrowth of the Unix and Plan 9 values of simplicity, pragmatism, economy, and ultimately compositionality.

Like Unix, the Plan 9 environment is really made for programmers and system operators. I'll leave a deeper explanation of what makes Plan 9 great for another post, but I do get into it a bit in the demo video for Wanix:

["Wanix: The Spirit of Plan 9 in Wasm" on YouTube](https://www.youtube.com/watch?v=kGBeT8lwbo0)

While I wanted to incorporate Plan 9 ideas into Wanix from the beginning, it wasn't until we rebuilt it from scratch with that intention that the magic really starts to come through. That's what this preview release is about.

Wanix is a whole new beast now. It's no longer a singular computing environment that runs in the browser. It's now a primitive for building environments in general. The demo shows a shell environment, but this environment is not the point. It's just a way to bootstrap Wanix so you can use and explore it interactively.

The point of this preview release is to get this primitive out there. I have my uses for Wanix, and I plan to share them with the final 0.3 release, but until then I wanted to let it all percolate. Maybe inspire people to get creative with their own use cases.

Here's a quick rundown of Wanix features in this release:

**Plan 9 inspired design**
With the original intention to enable exploring Plan 9 ideas on modern platforms, we've ended up with a radically simple architecture around per-process namespaces composed of file service capabilities using similar design patterns to those found in Plan 9.

**Single executable toolchain**
The `wanix` executable includes everything needed to produce Wanix environments.

**Filesystem is the only API**
The Wanix microkernel is now simply a VFS module with several built-in file services exposed via a standard filesystem API. This ends up making the module itself a file service.

**Built-in Linux shell**
Using the built-in file service primitives, Wanix can bootstrap a Linux-compatible shell based on Busybox. It comes with several helper commands for working with built-in file services.

**Tasks and namespaces**
The Wanix unit of compute is a task, which is equivalent and compatible with POSIX processes but allows for different execution strategies. Each task has its own "namespace," which is the customizable filesystem exposed to the task.

**Core file services**
Wanix includes two singleton file services: one to manage tasks (similar to procfs), and one to manage "capabilities" which are user allocated file services. Built-in capabilities include: tarfs, tmpfs, and loopback.

**Web file services**
With future non-browser deployments in mind, all web related file services are packaged in a web module, which is also built-in but not considered core. This module includes these work-in-progress file services:
* **opfs**: For working with the [OPFS](https://developer.mozilla.org/en-US/docs/Web/API/File_System_API/Origin_private_file_system) browser storage API
* **dom**: For inspecting and manipulating the DOM
* **worker**: For managing [web workers](https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API)
* **pickerfs**: Capability wrapping the `window.showDirectoryPicker()` method (not available yet in Safari and Firefox)
* **ws**: Capability for working with WebSocket connections
* **sw**: For configuring the [service worker](https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API), which is used by the system now to cache all resources needed to run Wanix allowing offline usage, as well as exposing virtual URLs to the root namespace.

Go programmers might also appreciate the filesystem toolkit we've been working on since before Wanix. It builds on the `fs.FS` abstraction in the standard library and gives you DSL-like utilities for defining virtual filesystems like the file services in Wanix. More on that in a dedicated post as well.

So far, the feedback has been really positive. I appreciate everybody taking the time to process it. There's still lots to do. Wanix is itself its own universe, but it's just one layer of the Tractor project. After a little vacation I'll be back to continue work on both fronts. As usual, I'd love help.

Speaking of help, shout-out to [Joël Franusic](https://joel.franusic.com/) for the help and support. And as usual, big thanks to my [GitHub sponsors](https://github.com/sponsors/progrium) for making this possible.
