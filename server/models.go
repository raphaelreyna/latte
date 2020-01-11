package server

import (
	"github.com/jinzhu/gorm"
	"time"
)

type Template struct {
	ID            uint64
	UID           string `gorm:"unique;not_null"`
	Data          []byte `gorm:"not_null"`
	ResourceCount uint8  `gorm:"not_null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Resource struct {
	ID          uint64
	TemplateUID string `gorm:"unique_index:uid_name;not_null"`
	Name        string `gorm:"unique_index:uid_name;not_null"`
	Data        []byte `gorm:"not_null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func GetTemplate(uid string, db *gorm.DB) (t *Template, err error) {
	err = db.Find(&t, "uid = ?", uid).Error
	return
}

func GetResources(tmplUID string, db *gorm.DB) (r []*Resource, err error) {
	err = db.Find(&r, "template_uid = ?", tmplUID).Error
	return
}
