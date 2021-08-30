package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	usersConnectCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_service_users_connect_count",
			Help: "The total amount of users that connect to the chat service",
		},
	)
	usersDisconnectCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_service_users_disconnect_count",
			Help: "The total amount of users that disconnect from the chat service",
		},
	)

	messageSentCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chat_service_message_sent_count",
			Help: "The total number of message sent to the chat service",
		},
	)
)

func metrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9002", nil)
}
