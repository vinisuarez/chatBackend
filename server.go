package main

import (
	"chatBackend/pb"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	natsConn   *nats.Conn
	redis      *redis.Client
	users      map[string]*pb.User
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	listOnline chan *Client
	broadcast  chan *pb.ChatMessage
}

const actionJoin string = "USER_JOIN"
const actionLeft string = "USER_LEFT"

const channelAdd string = "CHANNEL_ADD"
const channelRemove string = "CHANNEL_REMOVE"
const channelUser string = "CHANNEL_USER"

const onlinePrefix string = "onlineUser_"

func NewWebsocketServer(natsConn *nats.Conn, redis *redis.Client) *Server {
	server := &Server{
		natsConn:   natsConn,
		redis:      redis,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		listOnline: make(chan *Client),
		broadcast:  make(chan *pb.ChatMessage),
	}

	server.users = fecthOnlineUsers(redis)

	fmt.Printf("%v", server.users)

	return server
}

func fecthOnlineUsers(redisClient *redis.Client) map[string]*pb.User {
	iter := redisClient.Scan(ctx, 0, fmt.Sprintf("%s*", onlinePrefix), 0).Iterator()
	var users = make(map[string]*pb.User)
	for iter.Next(ctx) {
		var pbUser pb.User
		pbUserEncoded, _ := redisClient.Get(ctx, iter.Val()).Result()
		proto.Unmarshal([]byte(pbUserEncoded), &pbUser)
		users[pbUser.Name] = &pbUser
	}
	if err := iter.Err(); err != nil {
		log.Printf("Error on reading users from redis %s", err)
	}
	return users
}

func (server *Server) Run() {
	go server.listenMessage()
	for {
		select {

		case client := <-server.register:
			log.Println("server got client message for register")
			server.clients[client] = true

			msg := &pb.ChatMessage{
				FromUser: client.name,
				SentAt:   timestamppb.Now(),
				Text:     actionJoin,
			}
			server.publishChat(msg)

			pbUser := &pb.User{
				Id:   client.id.String(),
				Name: client.name,
			}
			server.publishChangeUser(pbUser, channelAdd)

			usersConnectCounter.Inc()

		case client := <-server.unregister:
			log.Println("server got client message for unregister")

			if _, ok := server.clients[client]; ok {
				delete(server.clients, client)
				close(client.send)
			}
			msg := &pb.ChatMessage{
				FromUser: client.name,
				SentAt:   timestamppb.Now(),
				Text:     actionLeft,
			}
			server.publishChat(msg)

			pbUser := &pb.User{
				Id:   client.id.String(),
				Name: client.name,
			}
			server.publishChangeUser(pbUser, channelRemove)
			usersDisconnectCounter.Inc()

		case client := <-server.listOnline:
			client.send <- server.buildOnlineUser()

		case message := <-server.broadcast:
			log.Println("server got client message broadcast")
			server.publishChat(message)
			messageSentCounter.Inc()
		}
	}
}

func (server *Server) buildOnlineUser() []byte {
	onlineUsers := make([]*pb.User, 0, len(server.users))
	for _, v := range server.users {
		onlineUsers = append(onlineUsers, v)
	}
	m := jsonpb.Marshaler{}
	js, _ := m.MarshalToString(&pb.OnlineUsers{
		OnlineUsers: onlineUsers,
	})
	return []byte(js)
}

func (server *Server) publishChat(msg *pb.ChatMessage) {
	m := jsonpb.Marshaler{}
	js, _ := m.MarshalToString(msg)
	server.natsConn.Publish(channelUser, []byte(js))
	server.natsConn.Flush()
}

func (server *Server) publishChangeUser(pbUser *pb.User, channel string) {
	pbUserEncoded, _ := proto.Marshal(pbUser)
	server.natsConn.Publish(channel, pbUserEncoded)
	server.natsConn.Flush()
}

func (server *Server) listenMessage() {
	server.natsConn.Subscribe(channelAdd, func(msg *nats.Msg) {
		var pbUser pb.User
		proto.Unmarshal(msg.Data, &pbUser)

		server.redis.Set(ctx, fmt.Sprintf("%s%s", onlinePrefix, pbUser.Name), msg.Data, 0)
		server.users[pbUser.Name] = &pbUser
		server.broadcastMessage(server.buildOnlineUser())
	})

	server.natsConn.Subscribe(channelRemove, func(msg *nats.Msg) {
		var pbUser pb.User
		proto.Unmarshal(msg.Data, &pbUser)

		server.redis.Del(ctx, fmt.Sprintf("%s%s", onlinePrefix, pbUser.Name))
		delete(server.users, pbUser.Name)
		server.broadcastMessage(server.buildOnlineUser())
	})

	server.natsConn.Subscribe(channelUser, func(msg *nats.Msg) {
		server.broadcastMessage(msg.Data)
	})
	server.natsConn.Flush()
}

func (server *Server) broadcastMessage(message []byte) {
	for client := range server.clients {
		client.send <- message
	}
}
