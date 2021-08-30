package main

import (
	"chatBackend/auth"
	"chatBackend/pb"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type AuthUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type API struct {
	redis *redis.Client
}

const loginKey string = "login_"
const userKey string = "user_"

func (api *API) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var user AuthUser

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := api.redis.Exists(ctx, fmt.Sprintf("%s%s", loginKey, user.Username)).Result()

	if err != nil {
		log.Fatal("Error fetching login user from redis")
	}

	if result > 0 {
		returnErrorResponse(w, "username already exists")
		return
	}

	hashedPass := auth.HashPassword([]byte(user.Password))

	if err != nil {
		returnErrorResponse(w, "password invalid")
		return
	}

	api.redis.Set(ctx, fmt.Sprintf("%s%s", loginKey, user.Username), hashedPass, 0)

	id, _ := uuid.NewRandom()
	pbUser := &pb.User{
		Id:   id.String(),
		Name: user.Username,
	}

	encodedUser, _ := proto.Marshal(pbUser)

	api.redis.Set(ctx, fmt.Sprintf("%s%s", userKey, user.Username), encodedUser, 0)

	w.Write([]byte("ok"))
}

func (api *API) HandleLogin(w http.ResponseWriter, r *http.Request) {

	var user AuthUser

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := api.redis.Get(ctx, fmt.Sprintf("%s%s", loginKey, user.Username)).Result()

	if err != nil {
		returnErrorResponse(w, "Wrong credentials")
		return
	}

	isValid := auth.ComparePasswords(result, []byte(user.Password))
	if !isValid || err != nil {
		returnErrorResponse(w, "Wrong credentials")
		return
	}

	result2, err := api.redis.Get(ctx, fmt.Sprintf("%s%s", userKey, user.Username)).Result()

	if err != nil {
		returnErrorResponse(w, "Wrong credentials")
		return
	}
	var pbUser pb.User
	proto.Unmarshal([]byte(result2), &pbUser)

	token, err := auth.CreateJWTToken(&pbUser)

	if err != nil {
		returnErrorResponse(w, "error creating jwt token")
		return
	}

	w.Write([]byte(token))

}

func returnErrorResponse(w http.ResponseWriter, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"status\": \"error\", \"reason\": \"%s\"}", reason)))
}
