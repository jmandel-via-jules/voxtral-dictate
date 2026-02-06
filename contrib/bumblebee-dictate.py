"""bumblebee-status module for voxtral-dictate

Shows "REC" with critical (red) styling when dictation is active.
Blank when idle or daemon not running.

Install:
    cp contrib/bumblebee-dictate.py ~/.config/bumblebee-status/modules/dictate.py

Then add "dictate" to your bumblebee-status module list in i3 config.
"""

import core.module
import core.widget
import core.decorators
import socket

class Module(core.module.Module):
    @core.decorators.every(seconds=1)
    def __init__(self, config, theme):
        super().__init__(config, theme, core.widget.Widget(self.status))
        self._sock = config.get("socket", "/tmp/dictate.sock")

    def status(self, widget):
        try:
            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.settimeout(0.5)
            s.connect(self._sock)
            s.sendall(b"status\n")
            resp = s.recv(64).decode().strip()
            s.close()
            if resp == "active":
                return "REC"
            return ""
        except:
            return ""

    def state(self, widget):
        if widget.full_text() == "REC":
            return ["critical"]
        return []
