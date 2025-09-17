package db

import "gorm.io/gorm"

type OauthProvider string

type ChatType string

const (
	Google OauthProvider = "google"

	PublicChat  ChatType = "public-chat"
	PrivateChat ChatType = "private-chat"
)

type Account struct {
	gorm.Model
	Username        string `json:"username" gorm:"not null"`
	Email           string `json:"email" gorm:"not null"`
	OauthProvider   string `json:"oauth_provider" gorm:"not null"`
	OauthProviderID string `json:"oauth_provider_id" gorm:"unique;not null"`
	TokenVersion    uint   `json:"token_version"`
}

type Message struct {
	gorm.Model
	SenderID   uint     `json:"sender_id"`
	Sender     Account  `json:"sender" gorm:"foreignKey:SenderID"`
	ReceiverID *uint    `json:"receiver_id"`
	Receiver   *Account `json:"receiver" gorm:"foreignKey:ReceiverID"`
	ChatType   ChatType `json:"chat_type"`
	Content    string   `json:"content"`
}
