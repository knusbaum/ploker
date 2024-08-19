# Ploker - Simple Planning Poker

Ploker provides a server and webasm-based web client for planning poker.

The server functions merely as a state-keeping and session-management service.
HTML rendering and logic exists mostly in the client.

In order to create a session, go to the home page and click `New Session`.

![image](https://github.com/user-attachments/assets/a5a93121-7692-4a72-95fc-d9d25130eee6)


This will generate a new session with a new session ID. To join an existing session, you can share the URL of the session with your teammates (i.e. `http://poker.fritterware.org/session/[your session ID]`)

![image](https://github.com/user-attachments/assets/8fc18892-1af3-4412-9bd5-f6a7ac369e1a)


Each teammate can vote with the number buttons. When everyone has voted, you may click `Reveal`, at which point everyone's scores will be revealed after a countdown.

![image](https://github.com/user-attachments/assets/43854477-0317-457f-a21b-c64bb00f92aa)
![image](https://github.com/user-attachments/assets/defaedf5-d3a1-49e8-a2ac-aad347bc6b9e)
![image](https://github.com/user-attachments/assets/3554fc9e-6a0a-4ace-b5d1-18d377847fad)


After the reveal, click the `Reset` button to start another round.

