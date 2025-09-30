package models

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type WebSocketConfig struct {
	Port string
}

type ServicesConfig struct {
	RideService           string
	DriverLocationService string
	AdminService          string
}

type Config struct {
	Database  DatabaseConfig
	RabbitMQ  RabbitMQConfig
	WebSocket WebSocketConfig
	Services  ServicesConfig
}
