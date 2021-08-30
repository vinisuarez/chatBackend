package auth

import (
	"chatBackend/pb"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const secret = "SomeStrongPass"

func CreateJWTToken(user *pb.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"Id":        user.GetId(),
		"Name":      user.GetName(),
		"ExpiresAt": time.Now().Unix() + 86400,
	})
	tokenString, err := token.SignedString([]byte(secret))

	return tokenString, err
}

type Claims struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	jwt.StandardClaims
}

func ValidateToken(tokenString string) (*pb.User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return &pb.User{
			Id:   claims.ID,
			Name: claims.Name,
		}, nil
	} else {
		return nil, err
	}
}
