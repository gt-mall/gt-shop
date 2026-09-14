package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
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
	var (
		initFlag  = flag.Bool("init", false, "run database migrations and seeds (clean install), then exit")
		seedFlag  = flag.Bool("seed", false, "run database seeds, then exit")
		alignFlag = flag.Bool("align", false, "align existing database schema_migrations to baseline and run seeds, then exit")
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of mall admin core:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  go run . [command|options]\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Commands:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  init        Run database migrations and seeds (clean install), then exit\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  seed        Run database seeds, then exit\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  align       Align existing database schema_migrations to baseline and run seeds, then exit\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	action := ""
	if *initFlag {
		action = "init"
	} else if *seedFlag {
		action = "seed"
	} else if *alignFlag {
		action = "align"
	} else if flag.NArg() > 0 {
		switch flag.Arg(0) {
		case "init":
			action = "init"
		case "seed":
			action = "seed"
		case "align":
			action = "align"
		default:
			flag.Usage()
			os.Exit(1)
		}
	}

	config := bootstrap.LoadConfig()

	if action != "" {
		if !config.DatabaseSet {
			log.Fatalf("DATABASE_DSN is not configured; cannot execute database %s", action)
		}
		db, err := bootstrap.NewDB(context.Background(), config)
		if err != nil {
			log.Fatalf("failed to connect to database for %s: %v", action, err)
		}
		sqlDB, dbErr := db.DB()
		if dbErr != nil {
			log.Fatalf("database handle is unavailable: %v", dbErr)
		}
		defer sqlDB.Close()

		ctx := context.Background()
		switch action {
		case "init":
			log.Print("starting database initialization (migrations + seeds)...")
			if err := bootstrap.RunInit(ctx, db); err != nil {
				log.Fatalf("database initialization failed: %v", err)
			}
			log.Print("database initialization completed successfully")
			return
		case "seed":
			log.Print("starting database seeding...")
			if err := bootstrap.RunSeeds(ctx, db); err != nil {
				log.Fatalf("database seeding failed: %v", err)
			}
			log.Print("database seeding completed successfully")
			return
		case "align":
			log.Print("starting database baseline alignment...")
			if err := bootstrap.AlignBaseline(ctx, db); err != nil {
				log.Fatalf("database baseline alignment failed: %v", err)
			}
			log.Print("database baseline alignment completed successfully")
			return
		}
	}
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
