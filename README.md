# Ploker - Simple Planning Poker

Ploker provides a server and webasm-based web client for planning poker.

The server functions merely as a state-keeping and session-management service.
HTML rendering and logic exists mostly in the client.

In order to create a session, go to the home page and click `New Session`. This will generate a new session with a new session ID. To join an existing session, you can share the URL of the session with your teammates (i.e. `http://poker.fritterware.org/session/[your session ID]`)

Each teammate can vote with the number buttons. When everyone has voted, you may click `Reveal`, at which point everyone's scores will be revealed after a countdown.

After the reveal, click the `Reset` button to start another round.

