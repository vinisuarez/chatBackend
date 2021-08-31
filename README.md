# chatBackend

Heyho.

Didn't have much time to do this, but at least i managed to get it working :)

The base of the server/client infra is based on gorilla web socket (https://github.com/gorilla/websocket)
also some code and ideas from https://github.com/jeroendk/go-vuejs-chat (like normal http api for auth users)

I want to make it different from other project and get out from my comfort zone and use nats instead of redis pub/sub, since you use this and i wanted to learn a bit about it.

Redis is used as DB to keep users credentials, map of user and token, and online users.


all services can be run using ```docker-compose up```

that should start nats, redis and 2 instance of the chat server on port 8080 and 8081.

you can test the application with http client and web sockets clients:


1 ) Register user: 

```	
curl -X POST \
  localhost:8080/api/register \
  -H 'Cache-Control: no-cache' \
  -H 'Content-Type: application/json' \
  -d '{ "username": "vini", "password": "1234"}'
```

2) login:

```
curl -X POST \
  localhost:8080/api/login \
  -H 'Cache-Control: no-cache' \
  -H 'Content-Type: application/json' \
  -d '{ "username": "vini", "password": "1234"}'
```

3) login should return a token what you should send in the web socket argument, e.i:
```
ws://localhost:8080/ws?bearer=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFeHBpcmVzQXQiOjE2MzAzNTgzODksIklkIjoiMzA2ODA5ZWItYTMyMi00Yzg0LWE1Y2QtNjMzM2MzNWIwMjZjIiwiTmFtZSI6InZpbmkifQ.EMzM-WVnwgmCG3S2SSt9A2rdwq8PnPDiDSMqTSwLU9c
``` 

4) You should get a list of online users when joining and when other users join or leave the server. Also you can get a list at any time by sending `!online`. Any other message sent will be considered a chat message.

