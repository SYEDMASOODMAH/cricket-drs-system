// Command edge-agent runs on venue capture hardware (Jetson-class device or
// ruggedized mini-PC). It captures video from USB-C tethered GoPro-class
// cameras (UVC webcam mode; HDMI+capture-card fallback), maintains a rolling
// buffer, and on trigger pushes the relevant window to the cloud Media
// Ingest Gateway. See docs/architecture.md Sections 2, 9a, and 10, and
// ADR-0003 for the camera hardware decision.
//
// Phase 1 scaffold: no capture/buffer logic implemented yet.
package main

import "log"

func main() {
	log.Println("edge-agent: scaffold only, no capture pipeline implemented (see docs/phases.md Phase 2)")
}
