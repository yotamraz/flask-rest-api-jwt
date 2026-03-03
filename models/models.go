package models

// User represents a registered user account.
type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string `gorm:"size:128;not null" json:"-"`
	Stores       []Store `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"stores,omitempty"`
}

// Store represents a store owned by a user.
type Store struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"size:64;not null" json:"name"`
	UserID uint   `gorm:"index;not null" json:"user_id"`
	Tags   []Tag  `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"tags,omitempty"`
	Items  []Item `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// Tag represents a tag associated with a store.
type Tag struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"size:64;not null" json:"name"`
	StoreID uint   `gorm:"index;not null" json:"store_id"`
}

// Item represents an item in a store.
type Item struct {
	ID      uint    `gorm:"primaryKey" json:"id"`
	Name    string  `gorm:"size:64;not null" json:"name"`
	Price   float64 `gorm:"not null" json:"price"`
	StoreID uint    `gorm:"index;not null" json:"store_id"`
}
