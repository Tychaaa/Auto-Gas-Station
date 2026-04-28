package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// adminAccounts хранит единственную пару логин/пароль для Basic Auth на /api/v1/admin/*
var adminAccounts gin.Accounts

// initAdminFromEnv читает ADMIN_USERNAME и ADMIN_PASSWORD из окружения
// Оба обязательны чтобы случайно не оставить админские ручки открытыми
func initAdminFromEnv() error {
	username := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" {
		return errors.New("ADMIN_USERNAME is required")
	}
	if password == "" {
		return errors.New("ADMIN_PASSWORD is required")
	}

	adminAccounts = gin.Accounts{username: password}
	log.Printf("admin auth configured for user %q", username)
	return nil
}

// adminAuth возвращает middleware Basic Auth с дополнительной проверкой что клиент пришел по loopback
// Это вторая линия защиты на случай если кто-то случайно сменит bind на 0.0.0.0
func adminAuth() gin.HandlerFunc {
	basicAuth := gin.BasicAuth(adminAccounts)

	return func(c *gin.Context) {
		if !isLoopbackClient(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "admin endpoints are available from loopback only",
			})
			return
		}
		basicAuth(c)
	}
}

// isLoopbackClient проверяет что запрос пришел с того же хоста
func isLoopbackClient(clientIP string) bool {
	switch clientIP {
	case "127.0.0.1", "::1":
		return true
	}
	return false
}
