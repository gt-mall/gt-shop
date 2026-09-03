package main

import (
	"context"
	"log"

	"mall/admin_core/bootstrap"
	_ "mall/admin_core/docs"
	"mall/admin_core/routers"
)

// @title Mall Admin Core API
// @version 0.1.0
// @description Simple commerce administration backend.
// @host localhost:8080
// @BasePath /
func main() {
	config := bootstrap.LoadConfig()

	pool, err := bootstrap.NewPool(context.Background(), config)
	if err != nil {
		log.Printf("database is unavailable; starting without database: %v", err)
	}
	if pool != nil {
		defer pool.Close()
		if config.AutoMigrate {
			if err := bootstrap.RunMigrations(context.Background(), pool, config.MigrationsDir); err != nil {
				log.Printf("migration failed: %v", err)
			}
		}
	}

	router := routers.NewRouter(config, pool)
	log.Printf("mall admin core listening on :%s", config.Port)
	if err := router.Run(":" + config.Port); err != nil {
		log.Fatal(err)
	}
}
