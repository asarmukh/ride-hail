package config

import (
	"bufio"
	"os"
	"ride-hail/internal/shared/models"
	"strings"
)

func LoadConfig(filename string) (*models.Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &models.Config{}
	var section string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.Contains(line, ":") {
			continue
		}

		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if strings.HasPrefix(val, "${") && strings.Contains(val, ":-") {
			inside := strings.TrimSuffix(strings.TrimPrefix(val, "${"), "}")
			val = strings.SplitN(inside, ":-", 2)[1]
		}

		switch section {
		case "database":
			switch key {
			case "host":
				cfg.Database.Host = val
			case "port":
				cfg.Database.Port = val
			case "user":
				cfg.Database.User = val
			case "password":
				cfg.Database.Password = val
			case "database":
				cfg.Database.Database = val
			}
		case "rabbitmq":
			switch key {
			case "host":
				cfg.RabbitMQ.Host = val
			case "port":
				cfg.RabbitMQ.Port = val
			case "user":
				cfg.RabbitMQ.User = val
			case "password":
				cfg.RabbitMQ.Password = val
			}
		case "websocket":
			if key == "port" {
				cfg.WebSocket.Port = val
			}
		case "services":
			switch key {
			case "ride_service":
				cfg.Services.RideService = val
			case "driver_location_service":
				cfg.Services.DriverLocationService = val
			case "admin_service":
				cfg.Services.AdminService = val
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cfg, nil
}
