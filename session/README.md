# Sessions

Sessions are MQTT sessions that will hold unsent (queued) mqtt packages. As mqtt can retransmit packages when connections are lost (when offline), these files are queued packages waiting to be sent.

You can delete session files manually, which will clear the session (delete all queued packages).