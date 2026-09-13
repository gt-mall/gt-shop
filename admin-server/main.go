package main

import (
	"context"
	"log"
	"time"

	"mall/admin_core/bootstrap"
	_ "mall/admin_core/docs"
	"mall/admin_core/internal/auth"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/routers"
)

// @title Mall Admin Core API
// @version 0.1.0
// @description Simple commerce administration backend.
// @host localhost:8100
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	config := bootstrap.LoadConfig()
	if config.JWTSecret == "" {
		log.Print("AUTH_JWT_SECRET is not configured; admin login is unavailable")
	}

	db, err := bootstrap.NewDB(context.Background(), config)
	if err != nil {
		log.Printf("database is unavailable; starting without database: %v", err)
	}
	var authorization *auth.Service
	casbinReady := true
	if db != nil {
		sqlDB, dbErr := db.DB()
		if dbErr != nil {
			log.Printf("database handle is unavailable; starting without database: %v", dbErr)
		} else {
			defer sqlDB.Close()
		}
		if config.AutoMigrate {
			if err := bootstrap.RunMigrations(context.Background(), db); err != nil {
				log.Printf("migration failed: %v", err)
				casbinReady = false
			}
		}
		if casbinReady {
			enforcer, err := bootstrap.NewCasbinEnforcer(db)
			if err != nil {
				log.Printf("casbin initialization failed: %v", err)
			} else {
				authorization = auth.NewService(db, enforcer, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)
			}
		}
	}
	if authorization == nil {
		authorization = auth.NewService(db, nil, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)
	}

	cacheInstance := bootstrap.NewCache(context.Background(), config)
	defer cacheInstance.Close()

	router := routers.NewRouter(config, db, authorization, cacheInstance)
	if db != nil && casbinReady {
		workerCtx, stopWorker := context.WithCancel(context.Background())
		defer stopWorker()
		go miniapppayment.NewService(db).RunTimeoutWorker(workerCtx,
			miniapppayment.DefaultOrderExpiry, miniapppayment.DefaultSweepInterval)
	}
	log.Printf("mall admin core listening on :%s", config.Port)
	if err := router.Run(":" + config.Port); err != nil {
		log.Fatal(err)
	}
}
