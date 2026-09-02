package model

type Application struct {
	BaseFields
	Application string
	Name        string
	Description string
	Enabled     bool
}
